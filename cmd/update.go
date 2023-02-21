package cmd

import (
	"github.com/calyptia/cli/cmd/agent"
	cnfg "github.com/calyptia/cli/cmd/config"
	"github.com/calyptia/cli/cmd/coreinstance"
	"github.com/calyptia/cli/cmd/endpoint"
	"github.com/calyptia/cli/cmd/environment"
	"github.com/calyptia/cli/cmd/pipeline"
	"github.com/calyptia/cli/cmd/project"
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
)

func newCmdUpdate(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		project.NewCmdUpdateProject(config),
		agent.NewCmdUpdateAgent(config),
		pipeline.NewCmdUpdatePipeline(config),
		pipeline.NewCmdUpdatePipelineSecret(config),
		pipeline.NewCmdUpdatePipelineFile(config),
		pipeline.NewCmdUpdatePipelineClusterObject(config),
		endpoint.NewCmdUpdateEndpoint(config),
		coreinstance.NewCmdUpdateCoreInstance(config),
		environment.NewCmdUpdateEnvironment(config),
		cnfg.NewCmdUpdateConfigSection(config),
		cnfg.NewCmdUpdateConfigSectionSet(config),
	)

	return cmd
}
