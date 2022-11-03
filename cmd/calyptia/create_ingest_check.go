package main

import (
	"context"
	"fmt"
	"github.com/calyptia/api/types"
	"github.com/spf13/cobra"
)

func newCmdCreateIngestCheck(config *config) *cobra.Command {
	var (
		retries         uint
		configSectionID string
		status          string
		environment     string
	)
	cmd := &cobra.Command{
		Use:   "ingest-check CORE_INSTANCE",
		Short: "Create an ingest check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			coreInstance := args[0]
			ctx := context.Background()
			if ok := types.ValidCheckStatus(types.CheckStatus(status)); !ok {
				return fmt.Errorf("invalid status: %s", status)
			}
			ingestCheck := types.CreateIngestCheck{
				Status:          types.CheckStatus(status),
				Retries:         retries,
				ConfigSectionID: configSectionID,
			}
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			coreInstanceID, err := config.loadAggregatorID(coreInstance, environmentID)
			check, err := config.cloud.CreateIngestCheck(ctx, coreInstanceID, ingestCheck)
			if err != nil {
				return err
			}
			cmd.Println(check.ID)
			return nil
		},
	}
	flags := cmd.Flags()
	flags.UintVar(&retries, "retires", 0, "number of retries")
	flags.StringVar(&configSectionID, "config-section-id", "", "config section ID")
	flags.StringVar(&status, "status", "", "status")
	flags.StringVar(&environment, "environment", "default", "calyptia environment name")
	return cmd
}
