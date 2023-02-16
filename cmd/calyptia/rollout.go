package main

import (
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/spf13/cobra"
)

func newCmdRollout(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout resources to previous versions",
	}

	cmd.AddCommand(
		newCmdRolloutPipeline(config),
	)

	return cmd
}
