package main

import "github.com/spf13/cobra"

func newCmdGet(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(
		newCmdGetMembers(config),
		newCmdGetAgents(config),
		newCmdGetAgent(config),
		newCmdGetAggregators(config),
		newCmdGetPipelines(config),
		newCmdGetPipeline(config),
		newCmdGetEndpoints(config),
		newCmdGetPipelineConfigHistory(config),
		newCmdGetPipelineStatusHistory(config),
		newCmdGetPipelineSecrets(config),
		newCmdGetPipelineFiles(config),
		newCmdGetPipelineFile(config),
		newCmdGetResourceProfiles(config),
		newCmdGetEnvironment(config),
	)

	return cmd
}
