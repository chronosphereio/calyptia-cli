package main

import "github.com/spf13/cobra"

func newCmdUpdate(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update aggregators, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdUpdateProject(config),
		newCmdUpdateAgent(config),
		newCmdUpdateAggregator(config),
		newCmdUpdatePipeline(config),
	)

	return cmd
}
