package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/agent"
	"github.com/calyptia/cli/commands/clusterobject"
	"github.com/calyptia/cli/commands/configsection"
	"github.com/calyptia/cli/commands/coreinstance"
	"github.com/calyptia/cli/commands/endpoint"
	"github.com/calyptia/cli/commands/environment"
	"github.com/calyptia/cli/commands/fleet"
	"github.com/calyptia/cli/commands/ingestcheck"
	"github.com/calyptia/cli/commands/members"
	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/commands/resourceprofile"
	"github.com/calyptia/cli/commands/tracerecord"
	"github.com/calyptia/cli/commands/tracesession"
	"github.com/calyptia/cli/config"
)

func newCmdGet(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(
		members.NewCmdGetMembers(cfg),
		agent.NewCmdGetAgents(cfg),
		agent.NewCmdGetAgent(cfg),
		coreinstance.NewCmdGetCoreInstances(cfg),
		coreinstance.NewCmdGetCoreInstanceFiles(cfg),
		coreinstance.NewCmdGetCoreInstanceSecrets(cfg),
		pipeline.NewCmdGetPipelines(cfg),
		pipeline.NewCmdGetPipeline(cfg),
		endpoint.NewCmdGetEndpoints(cfg),
		pipeline.NewCmdGetPipelineConfigHistory(cfg),
		pipeline.NewCmdGetPipelineStatusHistory(cfg),
		pipeline.NewCmdGetPipelineSecrets(cfg),
		pipeline.NewCmdGetPipelineFiles(cfg),
		pipeline.NewCmdGetPipelineFile(cfg),
		pipeline.NewCmdGetPipelineLog(cfg),
		pipeline.NewCmdGetPipelineLogs(cfg),
		clusterobject.NewCmdGetClusterObjects(cfg),
		pipeline.NewCmdGetPipelineClusterObjects(cfg),
		resourceprofile.NewCmdGetResourceProfiles(cfg),
		environment.NewCmdGetEnvironment(cfg),
		tracesession.NewCmdGetTraceSessions(cfg),
		tracesession.NewCmdGetTraceSession(cfg),
		tracerecord.NewCmdGetTraceRecords(cfg),
		configsection.NewCmdGetConfigSections(cfg),
		ingestcheck.NewCmdGetIngestChecks(cfg),
		ingestcheck.NewCmdGetIngestCheck(cfg),
		ingestcheck.NewCmdGetIngestCheckLogs(cfg),
		fleet.NewCmdGetFleets(cfg),
		fleet.NewCmdGetFleet(cfg),
		fleet.NewCmdGetFleetFiles(cfg),
		fleet.NewCmdGetFleetFile(cfg),
	)

	return cmd
}
