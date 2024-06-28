package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/agent"
	"github.com/calyptia/cli/commands/configsection"
	"github.com/calyptia/cli/commands/coreinstance"
	"github.com/calyptia/cli/commands/endpoint"
	"github.com/calyptia/cli/commands/environment"
	"github.com/calyptia/cli/commands/fleet"
	"github.com/calyptia/cli/commands/members"
	"github.com/calyptia/cli/commands/operator"
	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/commands/project"
	"github.com/calyptia/cli/config"
)

func newCmdUpdate(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		project.NewCmdUpdateProject(cfg),
		members.NewCmdUpdateMember(cfg),
		agent.NewCmdUpdateAgent(cfg),
		fleet.NewCmdUpdateFleet(cfg),
		fleet.NewCmdUpdateFleetFile(cfg),
		pipeline.NewCmdUpdatePipeline(cfg),
		pipeline.NewCmdUpdatePipelineSecret(cfg),
		pipeline.NewCmdUpdatePipelineFile(cfg),
		pipeline.NewCmdUpdatePipelineClusterObject(cfg),
		endpoint.NewCmdUpdateEndpoint(cfg),
		coreinstance.NewCmdUpdateCoreInstance(cfg),
		coreinstance.NewCmdUpdateCoreInstanceFile(cfg),
		coreinstance.NewCmdUpdateCoreInstanceSecret(cfg),
		environment.NewCmdUpdateEnvironment(cfg),
		configsection.NewCmdUpdateConfigSection(cfg),
		configsection.NewCmdUpdateConfigSectionSet(cfg),
		operator.NewCmdUpdate(),
	)

	return cmd
}
