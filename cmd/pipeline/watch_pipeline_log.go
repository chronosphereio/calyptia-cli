package pipeline

import (
	"fmt"
	"time"

	"github.com/chronosphereio/calyptia-cli/completer"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"

	cfg "github.com/chronosphereio/calyptia-cli/config"
)

func NewCmdWatchPipelineLogs(config *cfg.Config) *cobra.Command {
	var interval int
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "pipeline_log [pipeline name or id]",
		Short: "Get a specific pipeline log",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			pipelineKey := args[0] // Pipeline ID or name is the first argument

			pipelineID, err := completer.LoadPipelineID(pipelineKey)
			if err != nil {
				return
			}

			status := cloud.PipelineLogStatusDone
			watched := make(map[string]bool)

			// Function to fetch and display logs
			fetchLogs := func() {
				ff, err := config.Cloud.PipelineLogs(config.Ctx, cloud.ListPipelineLogs{
					PipelineID: pipelineID,
					Status:     &status,
				})

				if err != nil {
					fmt.Println("Error fetching pipeline logs:", err)
					return
				}

				for _, log := range ff.Items {
					if _, ok := watched[log.ID]; !ok {
						fmt.Println(log.Logs)
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
				case <-config.Ctx.Done():
					return
				}
			}
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&interval, "interval", "i", 5, "Interval in seconds to fetch the logs")

	return cmd
}
