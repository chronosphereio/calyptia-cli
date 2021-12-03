package main

import "github.com/spf13/cobra"

func newCmdRollout(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollout",
		Short: "Rollout resources to previous versions",
	}

	cmd.AddCommand(
		newCmdRolloutPipeline(config),
	)

	return cmd
}
