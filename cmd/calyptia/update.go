package main

import "github.com/spf13/cobra"

func newCmdUpdate(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdUpdateProject(config),
		newCmdUpdateAgent(config),
		newCmdUpdatePipeline(config),
		newCmdUpdatePipelineSecret(config),
		newCmdUpdatePipelineFile(config),
		newCmdUpdatePipelineClusterObject(config),
		newCmdUpdateEndpoint(config),
		newCmdUpdateCoreInstance(config),
		newCmdUpdateEnvironment(config),
		newCmdUpdateConfigSection(config),
		newCmdUpdateConfigSectionSet(config),
	)

	return cmd
}
