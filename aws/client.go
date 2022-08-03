package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"text/template"
	"time"

	"github.com/calyptia/core-images-index/go-index"
	"github.com/sethvargo/go-retry"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/calyptia/api/types"
)

const (
	userDataTemplate = `
CALYPTIA_CLOUD_PROJECT_TOKEN={{.ProjectToken}}
{{if .CoreInstanceName }}
CALYPTIA_CLOUD_AGGREGATOR_NAME={{.CoreInstanceName}}
{{end}}
{{if .CoreInstanceTags }}
CALYPTIA_CORE_INSTANCE_TAGS={{.CoreInstanceTags}}
{{end}}
{{if .CoreInstanceEnvironment }}
CALYPTIA_CORE_INSTANCE_ENVIRONMENT={{.CoreInstanceEnvironment}}
{{end}}
`
	instanceUpCheckTimeout = 10 * time.Minute
	instanceUpCheckBackOff = 5 * time.Second

	DefaultRegionName       = "us-east-1"
	DefaultInstanceTypeName = "t2.xlarge"
	coreInstanceTag         = "core-instance-name"
	securityGroupNameFormat = "%s-security-group"
	keyPairNameFormat       = "%s-key-pair"
)

var (
	instanceUpCheckMaxDuration = func() retry.Backoff {
		return retry.WithMaxDuration(instanceUpCheckTimeout, retry.NewConstant(instanceUpCheckBackOff))
	}
)

type (
	//go:generate moq -out client_mock.go . Client
	Client interface {
		EnsureKeyPair(ctx context.Context, keyPairName string) (string, error)
		FindMatchingAMI(ctx context.Context, region, version string) (string, error)
		EnsureInstanceType(ctx context.Context, instanceTypeName string) (string, error)
		EnsureSubnet(ctx context.Context, subNetID string) (string, error)
		EnsureSecurityGroup(ctx context.Context, securityGroupName, vpcID string) (string, error)
		EnsureSecurityGroupIngressRules(ctx context.Context, securityGroupID string) error
		EnsureAndAssociateElasticIPv4Address(ctx context.Context, instanceID, elasticIPv4AddressPool, elasticIPv4Address string) (string, error)
		CreateUserdata(in *CreateUserDataParams) (string, error)
		CreateInstance(ctx context.Context, in *CreateInstanceParams) (CreatedInstance, error)
		InstanceState(ctx context.Context, instanceID string) (string, error)
		DeleteKeyPair(ctx context.Context, keyPairName string) error
		DeleteSecurityGroup(ctx context.Context, securityGroupName string) error
		DeleteInstance(ctx context.Context, instanceID string) error
	}

	DefaultClient struct {
		Client
		client *ec2.Client
		prefix string
	}

	CreateUserDataParams struct {
		ProjectToken            string
		CoreInstanceName        string
		CoreInstanceTags        string
		CoreInstanceEnvironment string
	}

	ElasticIPAddressParams struct {
		Pool, Address string
	}

	CreateInstanceParams struct {
		Region,
		CoreInstanceName,
		CoreVersion,
		KeyPairName,
		SecurityGroupName,
		InstanceType,
		SubnetID string
		UserData        *CreateUserDataParams
		PublicIPAddress *ElasticIPAddressParams
	}

	CreatedInstance struct {
		CoreInstanceName string `json:"-"`
		types.MetadataAWS
	}
)

func (i *CreatedInstance) String() string {
	return fmt.Sprintf("instance-id: %s, instance-type: %s, privateIPv4: %s", i.EC2InstanceID, i.EC2InstanceType, i.PrivateIPv4)
}

func New(ctx context.Context, prefix, region, credentials, profileFile, profileName string) (*DefaultClient, error) {
	var opts []func(options *awsconfig.LoadOptions) error

	if region != "" {
		opts = append(opts, awsconfig.WithRegion(region))
	}

	if credentials != "" {
		opts = append(opts, awsconfig.WithSharedCredentialsFiles([]string{credentials}))
	}

	if profileFile != "" {
		opts = append(opts, awsconfig.WithSharedConfigFiles([]string{profileFile}))
	}

	if profileName != "" {
		opts = append(opts, awsconfig.WithSharedConfigProfile(profileName))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &DefaultClient{
		client: ec2.NewFromConfig(cfg),
		prefix: prefix,
	}, nil
}

func (c *DefaultClient) getTags(resourceType awstypes.ResourceType) []awstypes.TagSpecification {
	return []awstypes.TagSpecification{
		{
			ResourceType: resourceType,
			Tags: []awstypes.Tag{
				{
					Key:   aws.String(coreInstanceTag),
					Value: aws.String(c.prefix),
				},
			},
		},
	}
}

func (c *DefaultClient) InstanceState(ctx context.Context, instanceID string) (string, error) {
	describeInstanceStatus := &ec2.DescribeInstanceStatusInput{
		IncludeAllInstances: aws.Bool(true),
		InstanceIds:         []string{instanceID},
	}

	instanceStatus, err := c.client.DescribeInstanceStatus(ctx, describeInstanceStatus)
	if err != nil {
		return "", err
	}

	if len(instanceStatus.InstanceStatuses) == 0 {
		return "", ErrInstanceStatusNotFound
	}

	return string(instanceStatus.InstanceStatuses[0].InstanceState.Name), nil
}

func (c *DefaultClient) FindMatchingAMI(ctx context.Context, region, version string) (string, error) {
	containerIndex, err := index.NewContainer()
	if err != nil {
		return "", err
	}

	if version != "" {
		coreImageTag, err := containerIndex.Match(ctx, version)
		if err != nil {
			return "", err
		}
		version = coreImageTag
	} else {
		// If no specific core instance version has been provided, use the latest published image.
		latest, err := containerIndex.Last(ctx)
		if err != nil {
			return "", err
		}
		version = latest
	}

	awsIndex, err := index.NewAWS()
	if err != nil {
		return "", err
	}

	imageID, err := awsIndex.Match(ctx, region, version)
	if err != nil {
		return "", err
	}

	return imageID, nil
}

func (c *DefaultClient) EnsureAndAssociateElasticIPv4Address(ctx context.Context, instanceID, elasticIPv4AddressPool, elasticIPv4Address string) (string, error) {
	var allocateAddressInput ec2.AllocateAddressInput

	allocateAddressInput.TagSpecifications = c.getTags(awstypes.ResourceTypeElasticIp)

	if elasticIPv4Address != "" {
		allocateAddressInput.Address = &elasticIPv4Address
	}

	if elasticIPv4AddressPool != "" {
		allocateAddressInput.CustomerOwnedIpv4Pool = &elasticIPv4AddressPool
	}

	ipv4Allocation, err := c.client.AllocateAddress(ctx, &allocateAddressInput)
	if err != nil {
		return "", err
	}

	associateAddressInput := &ec2.AssociateAddressInput{
		AllocationId:       ipv4Allocation.AllocationId,
		AllowReassociation: aws.Bool(true),
		InstanceId:         aws.String(instanceID),
	}

	_, err = c.client.AssociateAddress(ctx, associateAddressInput)
	if err != nil {
		return "", err
	}

	return *ipv4Allocation.PublicIp, nil
}

func (c *DefaultClient) EnsureSubnet(ctx context.Context, subNetID string) (string, error) {
	var describeSubnetInput ec2.DescribeSubnetsInput

	// find the default subnet
	describeSubnetInput = ec2.DescribeSubnetsInput{
		Filters: []awstypes.Filter{
			{
				Name:   aws.String("subnet-id"),
				Values: []string{subNetID},
			},
		},
	}
	if subNetID == "" {
		describeSubnetInput = ec2.DescribeSubnetsInput{
			Filters: []awstypes.Filter{
				{
					Name:   aws.String("defaultForAz"),
					Values: []string{"true"},
				},
			},
		}
	}

	subNets, err := c.client.DescribeSubnets(ctx, &describeSubnetInput)
	if err != nil {
		return "", err
	}

	if len(subNets.Subnets) == 0 {
		return "", ErrSubnetNotFound
	}

	return *subNets.Subnets[0].VpcId, nil
}

func (c *DefaultClient) EnsureSecurityGroupIngressRules(ctx context.Context, securityGroupID string) error {
	authorizeSecurityGroupIngress := &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []awstypes.IpPermission{
			{
				FromPort: aws.Int32(0),
				// -1 means udp and tcp.
				IpProtocol: aws.String("-1"),
				IpRanges: []awstypes.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("allow all tcp/udp traffic to this calyptia-core instance"),
					},
				},
				ToPort: aws.Int32(65535),
			},
		},
	}
	_, err := c.client.AuthorizeSecurityGroupIngress(ctx, authorizeSecurityGroupIngress)
	if err != nil && !errorIsAlreadyExists(err) {
		return err
	}
	return nil
}

func (c *DefaultClient) EnsureSecurityGroup(ctx context.Context, securityGroupName, vpcID string) (string, error) {
	if securityGroupName == "" {
		securityGroupName = fmt.Sprintf(securityGroupNameFormat, c.prefix)
	}

	describeSecurityGroupsInput := &ec2.DescribeSecurityGroupsInput{
		GroupNames: []string{securityGroupName},
	}

	securityGroups, err := c.client.DescribeSecurityGroups(ctx, describeSecurityGroupsInput)
	if err != nil && !errorIsNotFound(err) {
		return "", err
	}

	if securityGroups != nil && len(securityGroups.SecurityGroups) > 0 {
		return *securityGroups.SecurityGroups[0].GroupId, nil
	}
	createSecurityGroupInput := &ec2.CreateSecurityGroupInput{
		Description:       aws.String(securityGroupName),
		GroupName:         aws.String(securityGroupName),
		VpcId:             aws.String(vpcID),
		TagSpecifications: c.getTags(awstypes.ResourceTypeSecurityGroup),
	}
	securityGroup, err := c.client.CreateSecurityGroup(ctx, createSecurityGroupInput)
	if err != nil {
		return "", err
	}
	return *securityGroup.GroupId, nil
}

func (c *DefaultClient) DeleteInstance(ctx context.Context, instanceID string) error {
	terminateInstancesInput := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}
	_, err := c.client.TerminateInstances(ctx, terminateInstancesInput)
	return err
}

func (c *DefaultClient) DeleteSecurityGroup(ctx context.Context, securityGroupName string) error {
	if securityGroupName == "" {
		securityGroupName = fmt.Sprintf(securityGroupNameFormat, c.prefix)
	}

	deleteSecurityGroupInput := &ec2.DeleteSecurityGroupInput{
		GroupName: aws.String(securityGroupName),
	}

	_, err := c.client.DeleteSecurityGroup(ctx, deleteSecurityGroupInput)
	return err
}

func (c *DefaultClient) DeleteKeyPair(ctx context.Context, keyPairName string) error {
	if keyPairName == "" {
		keyPairName = fmt.Sprintf(keyPairNameFormat, c.prefix)
	}

	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyPairName),
	}

	_, err := c.client.DeleteKeyPair(ctx, deleteKeyPairInput)
	return err
}

func (c *DefaultClient) EnsureKeyPair(ctx context.Context, keyPairName string) (string, error) {
	if keyPairName == "" {
		keyPairName = fmt.Sprintf(keyPairNameFormat, c.prefix)
	}

	createKeyPairInput := &ec2.CreateKeyPairInput{
		KeyName:           aws.String(keyPairName),
		TagSpecifications: c.getTags(awstypes.ResourceTypeKeyPair),
	}

	createKeyPair, err := c.client.CreateKeyPair(ctx, createKeyPairInput)
	if err != nil && errorIsAlreadyExists(err) {
		describeKeyPairInput := &ec2.DescribeKeyPairsInput{
			KeyNames: []string{keyPairName},
		}

		keyPairs, err := c.client.DescribeKeyPairs(ctx, describeKeyPairInput)
		if err != nil {
			return "", err
		}

		if len(keyPairs.KeyPairs) == 0 {
			return "", ErrKeyPairNotFound
		}

		return *keyPairs.KeyPairs[0].KeyName, nil
	}

	if err != nil {
		return "", err
	}

	return *createKeyPair.KeyName, nil
}

func (c *DefaultClient) EnsureInstanceType(ctx context.Context, instanceTypeName string) (string, error) {
	var out awstypes.InstanceType

	describeInstanceTypeInput := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []awstypes.InstanceType{awstypes.InstanceType(instanceTypeName)},
	}

	instanceTypes, err := c.client.DescribeInstanceTypes(ctx, describeInstanceTypeInput)
	if err != nil {
		return string(out), err
	}

	if len(instanceTypes.InstanceTypes) == 0 {
		return string(out), fmt.Errorf("could not find any results for instance type %s", instanceTypeName)
	}

	out = instanceTypes.InstanceTypes[0].InstanceType
	return string(out), nil
}

func (c DefaultClient) CreateUserdata(in *CreateUserDataParams) (string, error) {
	var out bytes.Buffer

	t, err := template.New("userdata").Parse(userDataTemplate)
	if err != nil {
		return "", err
	}

	err = t.Execute(&out, in)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(out.Bytes()), nil
}

func (c *DefaultClient) CreateInstance(ctx context.Context, params *CreateInstanceParams) (CreatedInstance, error) {
	var out CreatedInstance

	imageID, err := c.FindMatchingAMI(ctx, params.Region, params.CoreVersion)
	if err != nil {
		return out, fmt.Errorf("could not find a matching AMI for version %s on region %s: %w", params.CoreVersion, params.Region, err)
	}

	keyPairName, err := c.EnsureKeyPair(ctx, params.KeyPairName)
	if err != nil {
		return out, fmt.Errorf("could not find a suitable key pair for a key: %w", err)
	}

	instanceType, err := c.EnsureInstanceType(ctx, params.InstanceType)
	if err != nil {
		return out, fmt.Errorf("could not find a suitable instance type: %w", err)
	}

	vpcID, err := c.EnsureSubnet(ctx, params.SubnetID)
	if err != nil {
		return out, fmt.Errorf("could not find a suitable subnet: %w", err)
	}

	securityGroupID, err := c.EnsureSecurityGroup(ctx, params.SecurityGroupName, vpcID)
	if err != nil {
		return out, fmt.Errorf("could not find a suitable security group: %w", err)
	}

	err = c.EnsureSecurityGroupIngressRules(ctx, securityGroupID)
	if err != nil {
		return out, fmt.Errorf("could not apply ingress security rules: %w", err)
	}

	params.UserData.CoreInstanceName = params.CoreInstanceName

	userData, err := c.CreateUserdata(params.UserData)
	if err != nil {
		return out, fmt.Errorf("could not generate instance user data: %w", err)
	}

	runInstancesInput := &ec2.RunInstancesInput{
		MaxCount:          aws.Int32(1),
		MinCount:          aws.Int32(1),
		ImageId:           aws.String(imageID),
		InstanceType:      awstypes.InstanceType(instanceType),
		UserData:          aws.String(userData),
		SecurityGroupIds:  []string{securityGroupID},
		SubnetId:          aws.String(params.SubnetID),
		KeyName:           aws.String(keyPairName),
		TagSpecifications: c.getTags(awstypes.ResourceTypeInstance),
	}

	instances, err := c.client.RunInstances(ctx, runInstancesInput)
	if err != nil {
		return out, err
	}

	instance := instances.Instances[0]

	out.EC2InstanceID = *instance.InstanceId
	out.EC2InstanceType = string(instance.InstanceType)
	out.AMIID = *instance.ImageId
	out.Hostname = *instance.PublicDnsName
	out.PrivateIPv4 = *instance.PrivateIpAddress
	out.CoreInstanceName = params.CoreInstanceName

	if params.PublicIPAddress == nil {
		if instance.PublicIpAddress != nil {
			out.PublicIPv4 = *instance.PublicIpAddress
		}
		return out, nil
	}

	// await for the instance to reach available status, cannot be associated with an
	// IPv4 address if the instance is not ready.
	err = retry.Do(ctx, instanceUpCheckMaxDuration(), func(ctx context.Context) error {
		state, err := c.InstanceState(ctx, *instance.InstanceId)
		if err != nil {
			return retry.RetryableError(err)
		}

		if state != string(awstypes.InstanceStateNameRunning) {
			return retry.RetryableError(fmt.Errorf("instance not in running state"))
		}

		return nil
	})

	if err != nil {
		return out, err
	}

	elasticIpv4Address, err := c.EnsureAndAssociateElasticIPv4Address(ctx, out.EC2InstanceID,
		params.PublicIPAddress.Pool, params.PublicIPAddress.Address)

	if err != nil {
		return out, fmt.Errorf("could not associate public ipv4 address: %w", err)
	}
	out.PublicIPv4 = elasticIpv4Address
	return out, nil
}
