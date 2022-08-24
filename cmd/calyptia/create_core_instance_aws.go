package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/sethvargo/go-retry"

	"errors"

	"github.com/spf13/cobra"

	"github.com/calyptia/api/types"
	awsclient "github.com/calyptia/cli/aws"
)

const (
	coreInstanceUpCheckTimeout = 10 * time.Minute
	coreInstanceUpCheckBackoff = 5 * time.Second
)

var (
	errCoreInstanceNotFound        = errors.New("core instance not found")
	errCoreInstanceNotRunning      = errors.New("core instance not in running status")
	coreInstanceUpCheckMaxDuration = func() retry.Backoff {
		return retry.WithMaxDuration(coreInstanceUpCheckTimeout, retry.NewConstant(coreInstanceUpCheckBackoff))
	}
)

type (
	//go:generate moq -out core_instance_poller_mock.go . CoreInstancePoller
	CoreInstancePoller interface {
		Ready(ctx context.Context, environment, name string) (string, error)
	}

	DefaultCoreInstancePoller struct {
		CoreInstancePoller
		config *config
	}
)

func (c *DefaultCoreInstancePoller) Ready(ctx context.Context, environment, name string) (string, error) {
	var instance types.Aggregator

	params := types.AggregatorsParams{
		Name: &name,
	}

	if environment != "" {
		envs, err := c.config.cloud.Environments(ctx, c.config.projectID, types.EnvironmentsParams{
			Name: &environment,
		})
		if err != nil {
			return "", err
		}

		if len(envs.Items) == 0 {
			return "", fmt.Errorf("could not find environment with name: %s", environment)
		}

		params.EnvironmentID = &envs.Items[0].ID
	}

	err := retry.Do(ctx, coreInstanceUpCheckMaxDuration(), func(ctx context.Context) error {
		instances, err := c.config.cloud.Aggregators(ctx, c.config.projectID, params)

		if err != nil {
			return retry.RetryableError(err)
		}

		if len(instances.Items) == 0 {
			return retry.RetryableError(errCoreInstanceNotFound)
		}

		instance = instances.Items[0]
		if instance.Status != types.AggregatorStatusRunning {
			return retry.RetryableError(errCoreInstanceNotRunning)
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return instance.ID, nil
}

func newCmdCreateCoreInstanceOnAWS(config *config, client awsclient.Client, poller CoreInstancePoller) *cobra.Command {
	var (
		tags                   []string
		noHealthCheckPipeline  bool
		noElasticIPv4Address   bool
		debug                  bool
		coreInstanceVersion    string
		coreInstanceName       string
		environment            string
		credentials            string
		profileFile            string
		profileName            string
		region                 string
		subnetID               string
		keyPairName            string
		keyPairStorePath       string
		instanceTypeName       string
		securityGroupName      string
		elasticIPv4Address     string
		elasticIPv4AddressPool string
	)

	cmd := &cobra.Command{
		Use:     "aws",
		Aliases: []string{"ec2", "amazon"},
		Short:   "Setup a new core instance on Amazon EC2",
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				awsInstance awsclient.CreatedInstance
				err         error
			)

			ctx := context.Background()

			if client == nil {
				client, err = awsclient.New(ctx, coreInstanceName, region, credentials, profileFile, profileName, debug)
				if err != nil {
					return fmt.Errorf("could not initialize client: %w", err)
				}
			}

			if poller == nil {
				poller = &DefaultCoreInstancePoller{
					config: config,
				}
			}

			exists, err := coreInstanceNameExists(ctx, config, environment, coreInstanceName)
			if err != nil && !errors.Is(err, errCoreInstanceNotFound) {
				return fmt.Errorf("could not get core instance details from cloud API: %w", err)
			}

			if exists {
				return fmt.Errorf("core instance named \"%s\" already exists on environment %s, choose a different name", coreInstanceName, environment)
			}

			if keyPairName != "" {
				exists, err = client.KeyPairExists(ctx, keyPairName)
				if err != nil {
					return fmt.Errorf("could not find a suitable key pair for %s: %w", keyPairName, err)
				}
			}

			// the key hasn't been provided or it doesn't exists in the AWS keypair endpoint.
			if keyPairName == "" || !exists {
				var privateRSAKey string
				keyPairName, privateRSAKey, err = client.CreateKeyPair(ctx, keyPairName, environment)
				if err != nil {
					return fmt.Errorf("could not create a new key pair: %w", err)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "Writing private RSA key %s on path %s\n", keyPairName, keyPairStorePath)
				keyPath, err := writePrivateRSAKey(keyPairStorePath, keyPairName, []byte(privateRSAKey))
				if err != nil {
					return fmt.Errorf("could not store private RSA key: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote private RSA key on path %s\n", keyPath)
			}

			params := &awsclient.CreateInstanceParams{
				Region:            region,
				CoreVersion:       coreInstanceVersion,
				CoreInstanceName:  coreInstanceName,
				KeyPairName:       keyPairName,
				SecurityGroupName: securityGroupName,
				InstanceType:      instanceTypeName,
				SubnetID:          subnetID,
				Environment:       environment,
				UserData: &awsclient.CreateUserDataParams{
					ProjectToken: config.projectToken,
				},
			}

			if environment != "" {
				params.UserData.CoreInstanceEnvironment = environment
			}

			if tags != nil {
				params.UserData.CoreInstanceTags = strings.Join(tags, ",")
			}

			if !noElasticIPv4Address {
				params.PublicIPAddress = &awsclient.ElasticIPAddressParams{
					Pool:    elasticIPv4AddressPool,
					Address: elasticIPv4Address,
				}
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Creating calyptia core instance on AWS")
			awsInstance, err = client.CreateInstance(ctx, params)
			if err != nil {
				return fmt.Errorf("could not create AWS instance: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "calyptia core instance running on AWS %s\n", awsInstance.String())
			coreInstanceID, err := poller.Ready(ctx, environment, coreInstanceName)
			if err != nil {
				return fmt.Errorf("calyptia core instance not ready: %w", err)
			}

			metadata, err := json.Marshal(awsInstance)
			if err != nil {
				return fmt.Errorf("could not encode metadata: %w", err)
			}

			err = config.cloud.UpdateAggregator(ctx, coreInstanceID, types.UpdateAggregator{
				Metadata: (*json.RawMessage)(&metadata),
			})

			if err != nil {
				return fmt.Errorf("could not update metadata for core instance: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Calyptia core instance is ready to use.\n")
			return nil
		},
	}

	cwd, _ := os.Getwd()

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version (latest is the default)")
	fs.StringVar(&coreInstanceName, "name", "", "Core instance name (autogenerated if empty)")
	fs.BoolVar(&noHealthCheckPipeline, "no-health-check-pipeline", false, "Disable health check pipeline creation alongside the core instance")
	fs.StringVar(&environment, "environment", "default", "Calyptia environment name")
	fs.StringSliceVar(&tags, "tags", nil, "Tags to apply to the core instance.")
	fs.StringVar(&credentials, "credentials", "", "Path to the AWS credentials file. If not specified the default credential loader will be used.")
	fs.StringVar(&profileFile, "profile-file", "", "Path to the AWS profile file. If not specified the default credential loader will be used.")
	fs.StringVar(&profileName, "profile", "", "Name of the AWS profile to use, if not specified, the default profileFile will be used.")

	// Set of parameters that map into https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#RunInstancesInput
	fs.StringVar(&keyPairName, "key-pair", "", "AWS Key pair to use for SSH into the core instance.")
	fs.StringVar(&keyPairStorePath, "private-key-store-path", cwd, "Path to store the generated private SSH key (If no existing keys are provided).")
	fs.StringVar(&region, "region", awsclient.DefaultRegionName, "AWS region name to use in the instance.")
	fs.StringVar(&instanceTypeName, "instance-type", awsclient.DefaultInstanceTypeName, "AWS Instance type to use (see https://aws.amazon.com/es/ec2/instance-types/) for details.")
	fs.StringVar(&securityGroupName, "security-group", "", "AWS Security group name to use.")
	fs.StringVar(&subnetID, "subnet-id", "", "AWS subnet name to use.If you don't specify a subnet ID, we choose a default subnet from your default VPC for you. If you don't have a default VPC, you MUST specify a subnet.")
	fs.BoolVar(&noElasticIPv4Address, "no-elastic-ip", false, "Don't allocate a floating ip address for the instance.")
	fs.BoolVar(&debug, "debug", false, "Enable debug logging")
	fs.StringVar(&elasticIPv4Address, "elastic-ip", "", "IPv4 formatted address of an existing elastic ip address allocation to associate to this instance. If not provided, a new one will be allocated for the created VM.")
	fs.StringVar(&elasticIPv4AddressPool, "elastic-ip-address-pool", "", "IP address pool to allocate the elastic ip address from.")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)

	return cmd
}

func coreInstanceNameExists(ctx context.Context, config *config, environment, name string) (bool, error) {
	_, err := getCoreInstanceByName(ctx, config, environment, name)
	if err != nil {
		if errors.Is(err, errCoreInstanceNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func writePrivateRSAKey(dir, file string, contents []byte) (string, error) {
	keyFileName := path.Join(dir, file) + ".pem"
	err := os.WriteFile(keyFileName, contents, 0600)
	return keyFileName, err
}

func getCoreInstanceByName(ctx context.Context, config *config, environment, name string) (*types.Aggregator, error) {
	var out *types.Aggregator
	params := types.AggregatorsParams{
		Name: &name,
	}

	if environment != "" {
		envs, err := config.cloud.Environments(ctx, config.projectID, types.EnvironmentsParams{
			Name: &environment,
		})
		if err != nil {
			return out, err
		}

		if len(envs.Items) == 0 {
			return out, fmt.Errorf("could not find environment with name: %s", environment)
		}

		params.EnvironmentID = &envs.Items[0].ID
	}

	coreInstances, err := config.cloud.Aggregators(ctx, config.projectID, params)

	if err != nil {
		return out, err
	}

	if len(coreInstances.Items) == 0 {
		return out, errCoreInstanceNotFound
	}

	return &coreInstances.Items[0], nil
}
