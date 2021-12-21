package main

import "github.com/spf13/cobra"

func newCmdGet(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(
		newCmdGetProjects(config),
		newCmdGetAgents(config),
		newCmdGetAgent(config),
		newCmdGetAggregators(config),
		newCmdGetPipelines(config),
		newCmdGetPipeline(config),
		newCmdGetEndpoints(config),
		newCmdGetPipelineConfigHistory(config),
		newCmdGetPipelineStatusHistory(config),
		newCmdGetPipelineSecrets(config),
	)

	return cmd
}
