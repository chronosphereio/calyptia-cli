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
	cfg "github.com/calyptia/cli/config"
)

func newCmdDelete(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		agent.NewCmdDeleteAgents(config),
		agent.NewCmdDeleteAgent(config),
		fleet.NewCmdDeleteFleet(config),
		fleet.NewCmdDeleteFleetFile(config),
		pipeline.NewCmdDeletePipeline(config),
		pipeline.NewCmdDeletePipelines(config),
		endpoint.NewCmdDeleteEndpoint(config),
		pipeline.NewCmdDeletePipelineFile(config),
		pipeline.NewCmdDeletePipelineClusterObject(config),
		pipeline.NewCmdDeletePipelineLog(config),
		coreinstance.NewCmdDeleteCoreInstance(config),
		coreinstance.NewCmdDeleteCoreInstanceFile(config),
		coreinstance.NewCmdDeleteCoreInstanceSecret(config),
		coreinstance.NewCmdDeleteCoreInstances(config),
		environment.NewCmdDeleteEnvironment(config),
		tracesession.NewCmdDeleteTraceSession(config),
		configsection.NewCmdDeleteConfigSection(config),
		ingestcheck.NewCmdDeleteIngestCheck(config),
	)

	return cmd
}
