package aws

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	types2 "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"

	ifaces "github.com/calyptia/cli/aws/ifaces"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awstypes "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/sethvargo/go-retry"

	"github.com/calyptia/core-images-index/go-index"

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
{{if .CoreInstanceGitHubToken }}
GITHUB_TOKEN={{.CoreInstanceGitHubToken}}
{{end}}
{{if .CoreInstanceTLSVerify }}
CALYPTIA_CORE_TLS_VERIFY={{.CoreInstanceTLSVerify}}
{{end}}
{{if .SkipServiceCreation }}
CORE_INSTANCE_SKIP_SERVICE_CREATION={{.SkipServiceCreation}}
{{end}}
`
	instanceUpCheckTimeout = 10 * time.Minute
	instanceUpCheckBackOff = 5 * time.Second

	DefaultRegionName                 = "us-east-1"
	DefaultInstanceTypeName           = "t2.xlarge"
	DefaultCoreInstanceTag            = "core-instance-name"
	DefaultCoreInstanceEnvironmentTag = "environment"
	securityGroupNameFormat           = "%s-%s-sg"
	keyPairNameFormat                 = "%s-%s-key-pair"
)

var (
	instanceUpCheckMaxDuration = func() retry.Backoff {
		return retry.WithMaxDuration(instanceUpCheckTimeout, retry.NewConstant(instanceUpCheckBackOff))
	}
	instanceTerminateCheckMaxDuration = instanceUpCheckMaxDuration
)

type TagSpec map[string]string

type (
	//go:generate moq -rm -stub -out client_mock.go . Client
	Client interface {
		FindMatchingAMI(ctx context.Context, useTestImages bool, region, version string) (string, error)
		EnsureKeyPair(ctx context.Context, keyPairName, environment string) (string, error)
		EnsureInstanceType(ctx context.Context, instanceTypeName string) (string, error)
		EnsureSubnet(ctx context.Context, subNetID string) (string, error)
		EnsureSecurityGroup(ctx context.Context, securityGroupName, environment, vpcID string) (string, error)
		EnsureSecurityGroupIngressRules(ctx context.Context, securityGroupID string) error
		EnsureAndAssociateElasticIPv4Address(ctx context.Context, instanceID, environment, elasticIPv4AddressPool, elasticIPv4Address string) (string, error)
		CreateUserdata(in *CreateUserDataParams) (string, error)
		CreateInstance(ctx context.Context, in *CreateInstanceParams) (CreatedInstance, error)
		InstanceState(ctx context.Context, instanceID string) (string, error)
		DeleteKeyPair(ctx context.Context, keyPairID string) error
		DeleteSecurityGroup(ctx context.Context, securityGroupName string) error
		DeleteInstance(ctx context.Context, instanceID string) error
		GetResourcesByTags(ctx context.Context, tags TagSpec) ([]Resource, error)
		DeleteResources(ctx context.Context, resources []Resource) error
	}

	DefaultClient struct {
		Client
		ec2Client  ifaces.Client
		tagsClient *resourcegroupstaggingapi.Client
		prefix     string
	}

	CreateUserDataParams struct {
		ProjectToken            string
		CoreInstanceName        string
		CoreInstanceTags        string
		CoreInstanceEnvironment string
		CoreInstanceGitHubToken string
		CoreInstanceTLSVerify   string
		SkipServiceCreation     string
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
		Environment,
		SubnetID string
		UserData        *CreateUserDataParams
		PublicIPAddress *ElasticIPAddressParams
		UseTestImages   bool
	}

	CreatedInstance struct {
		CoreInstanceName string `json:"-"`
		types.MetadataAWS
	}
)

func (i *CreatedInstance) String() string {
	return fmt.Sprintf("instance-id: %s, instance-type: %s, privateIPv4: %s", i.EC2InstanceID, i.EC2InstanceType, i.PrivateIPv4)
}

func New(ctx context.Context, prefix, region, credentials, profileFile, profileName string, debug bool) (*DefaultClient, error) {
	var opts []func(options *awsconfig.LoadOptions) error

	if debug {
		opts = append(opts, awsconfig.WithClientLogMode(aws.LogSigning|aws.LogRequestWithBody|aws.LogResponseWithBody))
	}

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
		ec2Client:  ec2.NewFromConfig(cfg),
		tagsClient: resourcegroupstaggingapi.NewFromConfig(cfg),
		prefix:     prefix,
	}, nil
}

func (c *DefaultClient) getTags(resourceType awstypes.ResourceType, environment string) []awstypes.TagSpecification {
	return []awstypes.TagSpecification{
		{
			ResourceType: resourceType,
			Tags: []awstypes.Tag{
				{
					Key:   aws.String(DefaultCoreInstanceTag),
					Value: aws.String(c.prefix),
				},
				{
					Key:   aws.String(DefaultCoreInstanceEnvironmentTag),
					Value: aws.String(environment),
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

	instanceStatus, err := c.ec2Client.DescribeInstanceStatus(ctx, describeInstanceStatus)
	if err != nil {
		return "", err
	}

	if len(instanceStatus.InstanceStatuses) == 0 {
		return "", ErrInstanceStatusNotFound
	}

	return string(instanceStatus.InstanceStatuses[0].InstanceState.Name), nil
}

func (c *DefaultClient) FindMatchingAMI(ctx context.Context, useTestImages bool, region, version string) (string, error) {
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

	return awsIndex.Match(ctx, index.FilterOpts{
		Region:    region,
		Version:   version,
		TestIndex: useTestImages,
	})
}

func (c *DefaultClient) EnsureAndAssociateElasticIPv4Address(ctx context.Context, instanceID, environment, elasticIPv4AddressPool, elasticIPv4Address string) (string, error) {
	var allocateAddressInput ec2.AllocateAddressInput

	allocateAddressInput.TagSpecifications = c.getTags(awstypes.ResourceTypeElasticIp, environment)

	if elasticIPv4Address != "" {
		allocateAddressInput.Address = &elasticIPv4Address
	}

	if elasticIPv4AddressPool != "" {
		allocateAddressInput.CustomerOwnedIpv4Pool = &elasticIPv4AddressPool
	}

	ipv4Allocation, err := c.ec2Client.AllocateAddress(ctx, &allocateAddressInput)
	if err != nil {
		return "", err
	}

	associateAddressInput := &ec2.AssociateAddressInput{
		AllocationId:       ipv4Allocation.AllocationId,
		AllowReassociation: aws.Bool(true),
		InstanceId:         aws.String(instanceID),
	}

	_, err = c.ec2Client.AssociateAddress(ctx, associateAddressInput)
	if err != nil {
		return "", err
	}

	if ipv4Allocation.PublicIp == nil {
		return "", nil
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

	subNets, err := c.ec2Client.DescribeSubnets(ctx, &describeSubnetInput)
	if err != nil {
		return "", err
	}

	if len(subNets.Subnets) == 0 {
		return "", ErrSubnetNotFound
	}

	if subNets.Subnets[0].VpcId == nil {
		return "", nil
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
	_, err := c.ec2Client.AuthorizeSecurityGroupIngress(ctx, authorizeSecurityGroupIngress)
	if err != nil && !errorIsAlreadyExists(err) {
		return err
	}
	return nil
}

func (c *DefaultClient) EnsureSecurityGroup(ctx context.Context, securityGroupName, environment, vpcID string) (string, error) {
	if securityGroupName == "" {
		securityGroupName = fmt.Sprintf(securityGroupNameFormat, c.prefix, environment)
	}

	describeSecurityGroupsInput := &ec2.DescribeSecurityGroupsInput{
		GroupNames: []string{securityGroupName},
	}

	securityGroups, err := c.ec2Client.DescribeSecurityGroups(ctx, describeSecurityGroupsInput)
	if err != nil && !errorIsNotFound(err) {
		return "", err
	}

	if securityGroups != nil && len(securityGroups.SecurityGroups) > 0 {
		if securityGroups.SecurityGroups[0].GroupId == nil {
			return "", nil
		}

		return *securityGroups.SecurityGroups[0].GroupId, nil
	}

	createSecurityGroupInput := &ec2.CreateSecurityGroupInput{
		Description:       aws.String(securityGroupName),
		GroupName:         aws.String(securityGroupName),
		VpcId:             aws.String(vpcID),
		TagSpecifications: c.getTags(awstypes.ResourceTypeSecurityGroup, environment),
	}
	securityGroup, err := c.ec2Client.CreateSecurityGroup(ctx, createSecurityGroupInput)
	if err != nil {
		return "", err
	}

	if securityGroup.GroupId == nil {
		return "", nil
	}

	return *securityGroup.GroupId, nil
}

func (c *DefaultClient) ReleaseElasticIP(ctx context.Context, elasticIPID string) error {
	releaseAddressInput := &ec2.ReleaseAddressInput{
		AllocationId: aws.String(elasticIPID),
	}
	_, err := c.ec2Client.ReleaseAddress(ctx, releaseAddressInput)
	return err
}

func (c *DefaultClient) DeleteInstance(ctx context.Context, instanceID string) error {
	terminateInstancesInput := &ec2.TerminateInstancesInput{
		InstanceIds: []string{
			instanceID,
		},
	}
	_, err := c.ec2Client.TerminateInstances(ctx, terminateInstancesInput)
	return err
}

type Resource struct {
	Type awstypes.ResourceType
	ID   string
}

func (r *Resource) String() string {
	return fmt.Sprintf("Resource type: %s - Unique ID: %s", string(r.Type), r.ID)
}

func (c *DefaultClient) DeleteResources(ctx context.Context, resources []Resource) error {
	var (
		instancesIDs     []string
		securityGroupIDs []string
		elasticIPsIDs    []string
		keypairIDs       []string
	)

	for _, resource := range resources {
		switch resource.Type {
		case awstypes.ResourceTypeInstance:
			instancesIDs = append(instancesIDs, resource.ID)
		case awstypes.ResourceTypeKeyPair:
			keypairIDs = append(keypairIDs, resource.ID)
		case awstypes.ResourceTypeSecurityGroup:
			securityGroupIDs = append(securityGroupIDs, resource.ID)
		case awstypes.ResourceTypeElasticIp:
			elasticIPsIDs = append(elasticIPsIDs, resource.ID)
		}
	}

	for _, instanceID := range instancesIDs {
		err := c.DeleteInstance(ctx, instanceID)
		if err != nil {
			return err
		}

		// check if the instance switches to terminated status, in this way, all the
		// associated resources can be safely removed.
		err = retry.Do(ctx, instanceTerminateCheckMaxDuration(), func(ctx context.Context) error {
			state, err := c.InstanceState(ctx, instanceID)
			if err != nil {
				return retry.RetryableError(err)
			}

			if state != string(awstypes.InstanceStateNameTerminated) {
				return retry.RetryableError(fmt.Errorf("instance not in terminated state"))
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	for _, keypairID := range keypairIDs {
		err := c.DeleteKeyPair(ctx, keypairID)
		if err != nil {
			return err
		}
	}

	for _, elasticIPsID := range elasticIPsIDs {
		err := c.ReleaseElasticIP(ctx, elasticIPsID)
		if err != nil {
			return err
		}
	}

	for _, securityGroupID := range securityGroupIDs {
		err := c.DeleteSecurityGroup(ctx, securityGroupID)
		if err != nil {
			return err
		}
	}

	return nil

}

func (c *DefaultClient) GetResourcesByTags(ctx context.Context, tags TagSpec) ([]Resource, error) {
	var filters []types2.TagFilter
	var out []Resource

	for k, v := range tags {
		filters = append(filters, types2.TagFilter{
			Key:    &k,
			Values: []string{v},
		})
	}
	params := &resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: filters,
	}

	resources, err := c.tagsClient.GetResources(ctx, params)
	// Build the request with its input parameters
	if err != nil {
		return out, err
	}

	for _, resource := range resources.ResourceTagMappingList {
		splitById := strings.Split(*resource.ResourceARN, "/")
		splitByType := strings.Split(splitById[0], ":")

		if len(splitById) <= 0 || len(splitByType) <= 0 {
			continue
		}

		newResource := Resource{
			Type: awstypes.ResourceType(splitByType[len(splitByType)-1]),
			ID:   splitById[len(splitById)-1],
		}

		out = append(out, newResource)
	}

	return filterOutNonRunningResources(ctx, c, out), nil
}

// filterOutNonRunningResources given a list of resources, filter out of the array the ones that does not have a running
// state, for now is only instances but other resources might be stateful as well.
func filterOutNonRunningResources(ctx context.Context, client Client, resources []Resource) []Resource {
	var out []Resource

	for _, resource := range resources {
		switch resource.Type {
		case awstypes.ResourceTypeInstance:
			state, err := client.InstanceState(ctx, resource.ID)
			if err != nil {
				continue
			}
			if state == string(awstypes.InstanceStateNameRunning) {
				out = append(out, resource)
			}
		default:
			out = append(out, resource)
		}
	}
	return out
}

func (c *DefaultClient) GetInstanceById(ctx context.Context, instanceID string) ([]string, error) {
	var out []string
	describeInstanceInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	instances, err := c.ec2Client.DescribeInstances(ctx, describeInstanceInput)
	if err != nil {
		return out, err
	}

	if len(instances.Reservations) == 0 {
		return out, fmt.Errorf("not found instances with id: %s", instanceID)
	}

	for _, reservation := range instances.Reservations {
		for _, instance := range reservation.Instances {
			out = append(out, *instance.InstanceId)
		}
	}

	return out, nil
}

func (c *DefaultClient) DeleteSecurityGroup(ctx context.Context, securityGroupID string) error {
	deleteSecurityGroupInput := &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(securityGroupID),
	}

	_, err := c.ec2Client.DeleteSecurityGroup(ctx, deleteSecurityGroupInput)
	return err
}

func (c *DefaultClient) DeleteKeyPair(ctx context.Context, keyPairID string) error {
	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyPairId: aws.String(keyPairID),
	}
	_, err := c.ec2Client.DeleteKeyPair(ctx, deleteKeyPairInput)
	return err
}

func (c *DefaultClient) EnsureKeyPair(ctx context.Context, keyPairName, environment string) (string, error) {
	if keyPairName == "" {
		keyPairName = fmt.Sprintf(keyPairNameFormat, c.prefix, environment)
	}

	describeKeyPairInput := &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyPairName},
	}

	keyPairs, err := c.ec2Client.DescribeKeyPairs(ctx, describeKeyPairInput)
	if err != nil && !errorIsNotFound(err) {
		return "", err
	}

	if keyPairs != nil && len(keyPairs.KeyPairs) > 0 {
		if keyPairs.KeyPairs[0].KeyName == nil {
			return "", nil
		}

		return *keyPairs.KeyPairs[0].KeyName, nil
	}

	createKeyPairInput := &ec2.CreateKeyPairInput{
		KeyName:           aws.String(keyPairName),
		TagSpecifications: c.getTags(awstypes.ResourceTypeKeyPair, environment),
	}

	createdKeyPair, err := c.ec2Client.CreateKeyPair(ctx, createKeyPairInput)
	if err != nil {
		return "", err
	}

	if createdKeyPair.KeyName == nil {
		return "", nil
	}

	return *createdKeyPair.KeyName, nil
}

func (c *DefaultClient) EnsureInstanceType(ctx context.Context, instanceTypeName string) (string, error) {
	var out awstypes.InstanceType

	describeInstanceTypeInput := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []awstypes.InstanceType{awstypes.InstanceType(instanceTypeName)},
	}

	instanceTypes, err := c.ec2Client.DescribeInstanceTypes(ctx, describeInstanceTypeInput)
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

	imageID, err := c.FindMatchingAMI(ctx, params.UseTestImages, params.Region, params.CoreVersion)
	if err != nil {
		return out, fmt.Errorf("could not find a matching AMI for version %s on region %s: %w", params.CoreVersion, params.Region, err)
	}

	keyPairName, err := c.EnsureKeyPair(ctx, params.KeyPairName, params.Environment)
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

	securityGroupID, err := c.EnsureSecurityGroup(ctx, params.SecurityGroupName, params.Environment, vpcID)
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
		TagSpecifications: c.getTags(awstypes.ResourceTypeInstance, params.Environment),
	}

	instances, err := c.ec2Client.RunInstances(ctx, runInstancesInput)
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

	elasticIpv4Address, err := c.EnsureAndAssociateElasticIPv4Address(ctx, out.EC2InstanceID, params.Environment,
		params.PublicIPAddress.Pool, params.PublicIPAddress.Address)

	if err != nil {
		return out, fmt.Errorf("could not associate public ipv4 address: %w", err)
	}
	out.PublicIPv4 = elasticIpv4Address
	return out, nil
}
