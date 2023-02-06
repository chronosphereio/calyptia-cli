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
		newCmdGetCoreInstances(config),
		newCmdGetPipelines(config),
		newCmdGetPipeline(config),
		newCmdGetEndpoints(config),
		newCmdGetPipelineConfigHistory(config),
		newCmdGetPipelineStatusHistory(config),
		newCmdGetPipelineSecrets(config),
		newCmdGetPipelineFiles(config),
		newCmdGetPipelineFile(config),
		newCmdGetClusterObjects(config),
		newCmdGetPipelineClusterObjects(config),
		newCmdGetResourceProfiles(config),
		newCmdGetEnvironment(config),
		newCmdGetTraceSessions(config),
		newCmdGetTraceSession(config),
		newCmdGetTraceRecords(config),
		newCmdGetConfigSections(config),
		newCmdGetIngestChecks(config),
		newCmdGetIngestCheck(config),
	)

	return cmd
}
