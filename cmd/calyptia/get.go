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
		newCmdGetAggregators(config),
		newCmdGetAggregatorPipelines(config),
		newCmdGetPipelinePorts(config),
		newCmdGetPipelineConfigHistory(config),
		newCmdGetPipelineStatusHistory(config),
	)

	return cmd
}
