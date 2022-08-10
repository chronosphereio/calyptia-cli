package main

import "github.com/spf13/cobra"

func newCmdDelete(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete aggregators, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdDeleteAgents(config),
		newCmdDeleteAgent(config),
		newCmdDeleteAggregator(config),
		newCmdDeletePipeline(config),
		newCmdDeleteEndpoint(config),
		newCmdDeletePipelineFile(config),
		newCmdDeleteCoreInstance(config, nil),
		newCmdDeleteEnvironment(config),
		newCmdDeleteTraceSession(config),
	)

	return cmd
}
