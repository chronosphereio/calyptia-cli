package cmd

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/cmd/agent"
	"github.com/calyptia/cli/cmd/clusterobject"
	cnfg "github.com/calyptia/cli/cmd/config"
	"github.com/calyptia/cli/cmd/coreinstance"
	"github.com/calyptia/cli/cmd/endpoint"
	"github.com/calyptia/cli/cmd/environment"
	"github.com/calyptia/cli/cmd/fleet"
	"github.com/calyptia/cli/cmd/ingestcheck"
	"github.com/calyptia/cli/cmd/members"
	"github.com/calyptia/cli/cmd/pipeline"
	"github.com/calyptia/cli/cmd/resourceprofile"
	"github.com/calyptia/cli/cmd/tracerecord"
	"github.com/calyptia/cli/cmd/tracesession"
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
		clusterobject.NewCmdGetClusterObjects(config),
		pipeline.NewCmdGetPipelineClusterObjects(config),
		resourceprofile.NewCmdGetResourceProfiles(config),
		environment.NewCmdGetEnvironment(config),
		tracesession.NewCmdGetTraceSessions(config),
		tracesession.NewCmdGetTraceSession(config),
		tracerecord.NewCmdGetTraceRecords(config),
		cnfg.NewCmdGetConfigSections(config),
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
