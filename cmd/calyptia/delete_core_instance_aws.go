package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	awsclient "github.com/calyptia/cli/aws"
)

func newCmdDeleteCoreInstanceOnAWS(config *config, client awsclient.Client) *cobra.Command {
	var (
		debug       bool
		credentials string
		region      string
		profileFile string
		profileName string
		environment string
	)

	var skipError, confirmDelete bool

	cmd := &cobra.Command{
		Use:               "aws CORE_INSTANCE",
		Aliases:           []string{"ec2", "amazon"},
		Short:             "Delete a core instance from Amazon EC2",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeCoreInstances,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			coreInstanceName := args[0]

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			coreInstanceID, err := config.loadCoreInstanceID(coreInstanceName, environmentID)
			if !skipError && err != nil {
				return fmt.Errorf("could not load core instance ID: %w", err)
			}

			// TODO: Make sure to delete core instance from Cloud even if we cannot connect to AWS.

			ctx := cmd.Context()
			if client == nil {
				client, err = awsclient.New(ctx, coreInstanceName, region, credentials, profileFile, profileName, false)
				if err != nil {
					return fmt.Errorf("could not initialize AWS client: %w", err)
				}
			}

			exists, err := coreInstanceNameExists(ctx, config, environment, coreInstanceName)
			if err != nil {
				return fmt.Errorf("could not get core instance details from cloud API: %w", err)
			}

			if !exists {
				return fmt.Errorf("could not get core instance named %s on environment %s", coreInstanceName, environment)
			}

			itemsToDelete, err := client.GetResourcesByTags(ctx, awsclient.TagSpec{
				awsclient.DefaultCoreInstanceTag:            coreInstanceName,
				awsclient.DefaultCoreInstanceEnvironmentTag: environment,
			})

			if err != nil {
				return fmt.Errorf("could not get resources from AWS with the given tags: %w", err)
			}

			if len(itemsToDelete) == 0 {
				cmd.Println("nothing to delete")
				return nil
			}

			var toDelete []string
			for _, item := range itemsToDelete {
				toDelete = append(toDelete, item.String())
			}

			fmt.Fprintln(cmd.OutOrStdout(), "The following resources will be removed from your AWS account:\n"+strings.Join(toDelete, "\n"))

			if !confirmDelete {
				cmd.Print("You confirm the deletion of those resources? [y/N] ")
				confirmDelete, err = readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmDelete {
					cmd.Println("Aborted")
					return nil
				}
			}

			err = config.cloud.DeleteAggregator(ctx, coreInstanceID)
			if !skipError && err != nil {
				return err
			}

			err = client.DeleteResources(ctx, itemsToDelete)
			if !skipError && err != nil {
				return err
			}

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()

	fs.StringVar(&credentials, "credentials", "", "Path to the AWS credentials file. If not specified the default credential loader will be used.")
	fs.StringVar(&profileFile, "profile-file", "", "Path to the AWS profile file. If not specified the default credential loader will be used.")
	fs.StringVar(&profileName, "profile", "", "Name of the AWS profile to use, if not specified, the default profileFile will be used.")
	fs.StringVar(&region, "region", awsclient.DefaultRegionName, "AWS region name to use in the instance.")
	fs.StringVar(&environment, "environment", "default", "Calyptia environment name")
	fs.BoolVar(&skipError, "skip-error", false, "Skip errors during delete process")
	fs.BoolVarP(&confirmDelete, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.BoolVar(&debug, "debug", false, "Enable debug logging")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)

	return cmd
}
