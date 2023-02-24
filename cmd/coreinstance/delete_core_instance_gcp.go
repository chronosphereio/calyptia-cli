package coreinstance

import (
	"fmt"
	"os"
	"time"

	"github.com/calyptia/cli/gcp"

	rateLimiter "golang.org/x/time/rate"

	"github.com/spf13/cobra"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
)

const burstNumber = 1

func NewCmdDeleteCoreInstanceOnGCP(config *cfg.Config, client gcp.Client) *cobra.Command {
	var (
		environment string
		projectID   string
		credentials string
		rateLimit   time.Duration
	)
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "gcp CORE_INSTANCE",
		Aliases:           []string{"google", "gce"},
		Short:             "Delete a core instance from Google Compute Engine",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteCoreInstances,
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

			rateLimit := rateLimiter.Every(1 * time.Minute / rateLimit)
			limiter := rateLimiter.NewLimiter(rateLimit, burstNumber)
			for {
				if err := limiter.Wait(ctx); err != nil {
					return err
				}

				operation, err := client.FollowOperations(ctx)
				if err != nil {
					return err
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
	fs.DurationVar(&rateLimit, "request-per-minute", 20, "Rate limit for operations")

	_ = fs.MarkHidden("request-per-minute")

	return cmd
}
