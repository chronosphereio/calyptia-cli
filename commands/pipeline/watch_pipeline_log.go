package pipeline

import (
	"time"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdWatchPipelineLogs(cfg *config.Config) *cobra.Command {
	var interval int

	cmd := &cobra.Command{
		Use:   "pipeline_log [pipeline name or id]",
		Short: "Get a specific pipeline log",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			pipelineKey := args[0] // Pipeline ID or name is the first argument

			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return
			}

			status := cloudtypes.PipelineLogStatusDone
			watched := make(map[string]bool)

			// Function to fetch and display logs
			fetchLogs := func() {
				ff, err := cfg.Cloud.PipelineLogs(ctx, cloudtypes.ListPipelineLogs{
					PipelineID: pipelineID,
					Status:     &status,
				})

				if err != nil {
					cmd.PrintErrf("Error fetching pipeline logs: %v\n", err)
					return
				}

				for _, log := range ff.Items {
					if _, ok := watched[log.ID]; !ok {
						cmd.Println(log.Logs)
						watched[log.ID] = true
					}
				}
			}

			// Immediate log fetching if interval is not set
			if interval <= 0 {
				fetchLogs()
				return
			}

			// Fetch logs every 'interval' seconds, similar to Unix watch
			ticker := time.NewTicker(time.Duration(interval) * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					fetchLogs()
				case <-ctx.Done():
					return
				}
			}
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&interval, "interval", "i", 5, "Interval in seconds to fetch the logs")

	return cmd
}
