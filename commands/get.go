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
	cfg "github.com/calyptia/cli/config"
)

func newCmdGet(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Display one or many resources",
	}

	cmd.AddCommand(
		members.NewCmdGetMembers(config),
		agent.NewCmdGetAgents(config),
		agent.NewCmdGetAgent(config),
		coreinstance.NewCmdGetCoreInstances(config),
		coreinstance.NewCmdGetCoreInstanceFiles(config),
		coreinstance.NewCmdGetCoreInstanceSecrets(config),
		pipeline.NewCmdGetPipelines(config),
		pipeline.NewCmdGetPipeline(config),
		endpoint.NewCmdGetEndpoints(config),
		pipeline.NewCmdGetPipelineConfigHistory(config),
		pipeline.NewCmdGetPipelineStatusHistory(config),
		pipeline.NewCmdGetPipelineSecrets(config),
		pipeline.NewCmdGetPipelineFiles(config),
		pipeline.NewCmdGetPipelineFile(config),
		pipeline.NewCmdGetPipelineLog(config),
		pipeline.NewCmdGetPipelineLogs(config),
		clusterobject.NewCmdGetClusterObjects(config),
		pipeline.NewCmdGetPipelineClusterObjects(config),
		resourceprofile.NewCmdGetResourceProfiles(config),
		environment.NewCmdGetEnvironment(config),
		tracesession.NewCmdGetTraceSessions(config),
		tracesession.NewCmdGetTraceSession(config),
		tracerecord.NewCmdGetTraceRecords(config),
		configsection.NewCmdGetConfigSections(config),
		ingestcheck.NewCmdGetIngestChecks(config),
		ingestcheck.NewCmdGetIngestCheck(config),
		ingestcheck.NewCmdGetIngestCheckLogs(config),
		fleet.NewCmdGetFleets(config),
		fleet.NewCmdGetFleet(config),
		fleet.NewCmdGetFleetFiles(config),
		fleet.NewCmdGetFleetFile(config),
	)

	return cmd
}
