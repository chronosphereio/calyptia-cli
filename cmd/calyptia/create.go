package main

import "github.com/spf13/cobra"

func newCmdCreate(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create aggregators, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdCreatePipeline(config),
		newCmdCreateResourceProfile(config),
		newCmdCreatePipelineFile(config),
	)

	return cmd
}
