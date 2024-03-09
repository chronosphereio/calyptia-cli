package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/cmd/pipeline"
	cfg "github.com/chronosphereio/calyptia-cli/config"
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
