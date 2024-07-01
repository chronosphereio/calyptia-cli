package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/config"
)

func newCmdRollout(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout resources to previous versions",
	}

	cmd.AddCommand(
		pipeline.NewCmdRolloutPipeline(cfg),
	)

	return cmd
}
