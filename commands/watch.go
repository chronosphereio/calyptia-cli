package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/config"
)

func newCmdWatch(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "watch for events or logs",
	}

	cmd.AddCommand(
		pipeline.NewCmdWatchPipelineLogs(cfg),
	)

	return cmd
}
