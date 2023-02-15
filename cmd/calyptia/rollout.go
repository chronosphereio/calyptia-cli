package main

import (
	"github.com/calyptia/cli/cmd/calyptia/utils"
	"github.com/spf13/cobra"
)

func newCmdRollout(config *utils.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout resources to previous versions",
	}

	cmd.AddCommand(
		newCmdRolloutPipeline(config),
	)

	return cmd
}
