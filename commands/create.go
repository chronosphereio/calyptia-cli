package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/configsection"
	"github.com/calyptia/cli/commands/coreinstance"
	"github.com/calyptia/cli/commands/environment"
	"github.com/calyptia/cli/commands/fleet"
	"github.com/calyptia/cli/commands/ingestcheck"
	"github.com/calyptia/cli/commands/invitation"
	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/commands/resourceprofile"
	"github.com/calyptia/cli/commands/tracesession"
	"github.com/calyptia/cli/config"
)

func newCmdCreate(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		invitation.NewCmdSendInvitation(cfg),
		coreinstance.NewCmdCreateCoreInstance(cfg),
		coreinstance.NewCmdCreateCoreInstanceFile(cfg),
		coreinstance.NewCmdCreateCoreInstanceSecret(cfg),
		pipeline.NewCmdCreatePipeline(cfg),
		resourceprofile.NewCmdCreateResourceProfile(cfg),
		pipeline.NewCmdCreatePipelineFile(cfg),
		pipeline.NewCmdCreatePipelineLog(cfg),
		environment.NewCmdCreateEnvironment(cfg),
		tracesession.NewCmdCreateTraceSession(cfg),
		configsection.NewCmdCreateConfigSection(cfg),
		ingestcheck.NewCmdCreateIngestCheck(cfg),
		fleet.NewCmdCreateFleet(cfg),
		fleet.NewCmdCreateFleetFile(cfg),
	)

	return cmd
}
