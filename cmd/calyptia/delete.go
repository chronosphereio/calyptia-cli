package main

import (
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/spf13/cobra"
)

func newCmdDelete(config *cfg.Config) *cobra.Command {
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
