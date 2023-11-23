package cmd

import (
	"github.com/calyptia/cli/cmd/pipeline"
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
)

func newCmdWatch(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "watch for events or logs",
	}

	cmd.AddCommand(
		pipeline.NewCmdWatchPipelineLogs(config),
	)

	return cmd
}
