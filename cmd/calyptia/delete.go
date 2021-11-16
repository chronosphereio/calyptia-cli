package main

import "github.com/spf13/cobra"

func newCmdDelete(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete aggregators, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdDeleteAggregator(config),
		newCmdDeletePipeline(config),
	)

	return cmd
}
