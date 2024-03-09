package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/cmd/agent"
	cnfg "github.com/chronosphereio/calyptia-cli/cmd/config"
	"github.com/chronosphereio/calyptia-cli/cmd/coreinstance"
	"github.com/chronosphereio/calyptia-cli/cmd/endpoint"
	"github.com/chronosphereio/calyptia-cli/cmd/environment"
	"github.com/chronosphereio/calyptia-cli/cmd/fleet"
	"github.com/chronosphereio/calyptia-cli/cmd/members"
	"github.com/chronosphereio/calyptia-cli/cmd/operator"
	"github.com/chronosphereio/calyptia-cli/cmd/pipeline"
	"github.com/chronosphereio/calyptia-cli/cmd/project"
	cfg "github.com/chronosphereio/calyptia-cli/config"
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
		cnfg.NewCmdUpdateConfigSection(config),
		cnfg.NewCmdUpdateConfigSectionSet(config),
		operator.NewCmdUpdate(),
	)

	return cmd
}
