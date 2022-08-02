package main

import "github.com/spf13/cobra"

func newCmdCreate(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create aggregators, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdCreateCoreInstance(config),
		newCmdCreatePipeline(config),
		newCmdCreateResourceProfile(config),
		newCmdCreatePipelineFile(config),
		newCmdCreateEnvironment(config),
	)

	return cmd
}
