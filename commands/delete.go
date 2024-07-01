package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/agent"
	"github.com/calyptia/cli/commands/configsection"
	"github.com/calyptia/cli/commands/coreinstance"
	"github.com/calyptia/cli/commands/endpoint"
	"github.com/calyptia/cli/commands/environment"
	"github.com/calyptia/cli/commands/fleet"
	"github.com/calyptia/cli/commands/ingestcheck"
	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/commands/tracesession"
	"github.com/calyptia/cli/config"
)

func newCmdDelete(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		agent.NewCmdDeleteAgents(cfg),
		agent.NewCmdDeleteAgent(cfg),
		fleet.NewCmdDeleteFleet(cfg),
		fleet.NewCmdDeleteFleetFile(cfg),
		pipeline.NewCmdDeletePipeline(cfg),
		pipeline.NewCmdDeletePipelines(cfg),
		endpoint.NewCmdDeleteEndpoint(cfg),
		pipeline.NewCmdDeletePipelineFile(cfg),
		pipeline.NewCmdDeletePipelineClusterObject(cfg),
		pipeline.NewCmdDeletePipelineLog(cfg),
		coreinstance.NewCmdDeleteCoreInstance(cfg),
		coreinstance.NewCmdDeleteCoreInstanceFile(cfg),
		coreinstance.NewCmdDeleteCoreInstanceSecret(cfg),
		coreinstance.NewCmdDeleteCoreInstances(cfg),
		environment.NewCmdDeleteEnvironment(cfg),
		tracesession.NewCmdDeleteTraceSession(cfg),
		configsection.NewCmdDeleteConfigSection(cfg),
		ingestcheck.NewCmdDeleteIngestCheck(cfg),
	)

	return cmd
}
