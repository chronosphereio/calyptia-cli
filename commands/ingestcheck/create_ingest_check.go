package ingestcheck

import (
	"fmt"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdCreateIngestCheck(cfg *config.Config) *cobra.Command {
	var (
		retries         uint
		configSectionID string
		status          string
		collectLogs     bool
	)

	cmd := &cobra.Command{
		Use:   "ingest_check CORE_INSTANCE",
		Short: "Create an ingest check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			coreInstance := args[0]

			params := cloudtypes.CreateIngestCheck{
				CollectLogs: collectLogs,
			}
			if configSectionID == "" {
				return fmt.Errorf("invalid config section id")
			}

			params.ConfigSectionID = configSectionID
			if retries > 0 {
				params.Retries = retries
			}

			if status != "" && !cloudtypes.ValidCheckStatus(cloudtypes.CheckStatus(status)) {
				return fmt.Errorf("invalid check status")
			}

			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstance)
			if err != nil {
				return err
			}

			check, err := cfg.Cloud.CreateIngestCheck(ctx, coreInstanceID, params)
			if err != nil {
				return err
			}
			cmd.Println(check.ID)
			return nil
		},
	}
	flags := cmd.Flags()
	flags.UintVar(&retries, "retries", 0, "number of retries")
	flags.StringVar(&configSectionID, "config-section-id", "", "config section ID")
	flags.StringVar(&status, "status", "", "status")
	flags.BoolVar(&collectLogs, "collect-logs", false, "Collect logs from the kubernetes pods once the job is finished")
	return cmd
}
