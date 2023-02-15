package main

import (
	"github.com/calyptia/cli/cmd/calyptia/utils"
	"github.com/spf13/cobra"
)

func newCmdDelete(config *utils.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdDeleteAgents(config),
		newCmdDeleteAgent(config),
		newCmdDeletePipeline(config),
		newCmdDeletePipelines(config),
		newCmdDeleteEndpoint(config),
		newCmdDeletePipelineFile(config),
		newCmdDeletePipelineClusterObject(config),
		newCmdDeleteCoreInstance(config, nil),
		newCmdDeleteCoreInstances(config),
		newCmdDeleteEnvironment(config),
		newCmdDeleteTraceSession(config),
		newCmdDeleteConfigSection(config),
		newCmdDeleteIngestCheck(config),
	)

	return cmd
}
