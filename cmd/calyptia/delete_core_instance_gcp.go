package main

import (
	"fmt"
	"os"

	"github.com/calyptia/cli/gcp"

	"github.com/spf13/cobra"
)

func newCmdDeleteCoreInstanceOnGCP(config *config, client gcp.Client) *cobra.Command {
	var (
		environment string
		projectID   string
		credentials string
	)
	cmd := &cobra.Command{
		Use:               "gcp CORE_INSTANCE",
		Aliases:           []string{"google", "gce"},
		Short:             "Delete a core instance from Google Compute Engine",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAggregators,
		RunE: func(cmd *cobra.Command, args []string) error {
			coreInstanceName := args[0]
			ctx := cmd.Context()
			if client == nil {
				var err error
				client, err = gcp.New(ctx, projectID, environment, credentials)
				if err != nil {
					return fmt.Errorf("could not initialize GCP client: %w", err)
				}
			}

			err := client.Delete(ctx, coreInstanceName)
			if err != nil {
				return fmt.Errorf("could not delete core instance: %w", err)
			}
			_, err = client.FollowOperations(ctx)
			if err != nil {
				return fmt.Errorf("could not get operation: %w", err)
			}

			cmd.Printf("[*] Waiting for delete operation...")

			for {
				operation, err := client.FollowOperations(ctx)

				if err != nil || operation.Error != nil {
					cmd.PrintErrf("an error occurred with the operation %s", operation.Name)
					return nil
				}

				if operation.Status == OperationConcluded {
					cmd.Println("done.")
					break
				}
			}

			cmd.Printf("[*] The instance %s has been deleted", coreInstanceName)

			return nil
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&projectID, "project-id", "", "GCP project ID")
	fs.StringVar(&environment, "environment", "default", "Calyptia environment name")
	fs.StringVar(&credentials, "credentials", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), "Path to GCP credentials file. (default is $GOOGLE_APPLICATION_CREDENTIALS)")
	return cmd
}
