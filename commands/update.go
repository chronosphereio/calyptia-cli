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
	cfg "github.com/calyptia/cli/config"
)

func newCmdUpdate(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		project.NewCmdUpdateProject(config),
		members.NewCmdUpdateMember(config),
		agent.NewCmdUpdateAgent(config),
		fleet.NewCmdUpdateFleet(config),
		fleet.NewCmdUpdateFleetFile(config),
		pipeline.NewCmdUpdatePipeline(config),
		pipeline.NewCmdUpdatePipelineSecret(config),
		pipeline.NewCmdUpdatePipelineFile(config),
		pipeline.NewCmdUpdatePipelineClusterObject(config),
		endpoint.NewCmdUpdateEndpoint(config),
		coreinstance.NewCmdUpdateCoreInstance(config),
		coreinstance.NewCmdUpdateCoreInstanceFile(config),
		coreinstance.NewCmdUpdateCoreInstanceSecret(config),
		environment.NewCmdUpdateEnvironment(config),
		configsection.NewCmdUpdateConfigSection(config),
		configsection.NewCmdUpdateConfigSectionSet(config),
		operator.NewCmdUpdate(),
	)

	return cmd
}
