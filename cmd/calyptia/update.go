package main

import "github.com/spf13/cobra"

func newCmdUpdate(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update aggregators, pipelines, core instances etc.",
	}

	cmd.AddCommand(
		newCmdUpdateProject(config),
		newCmdUpdateAgent(config),
		newCmdUpdatePipeline(config),
		newCmdUpdatePipelineSecret(config),
		newCmdUpdatePipelineFile(config),
		newCmdUpdateEndpoint(config),
		newCmdUpdateCoreInstance(config),
		newCmdUpdateEnvironment(config),
	)

	return cmd
}
