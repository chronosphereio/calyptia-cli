package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/cmd/pipeline"
	cfg "github.com/chronosphereio/calyptia-cli/config"
)

func newCmdRollout(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout resources to previous versions",
	}

	cmd.AddCommand(
		pipeline.NewCmdRolloutPipeline(config),
	)

	return cmd
}
