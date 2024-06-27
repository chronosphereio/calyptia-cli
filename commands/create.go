package commands

import (
	"github.com/spf13/cobra"

	cnfg "github.com/calyptia/cli/commands/config"
	"github.com/calyptia/cli/commands/coreinstance"
	"github.com/calyptia/cli/commands/environment"
	"github.com/calyptia/cli/commands/fleet"
	"github.com/calyptia/cli/commands/ingestcheck"
	"github.com/calyptia/cli/commands/invitation"
	"github.com/calyptia/cli/commands/pipeline"
	"github.com/calyptia/cli/commands/resourceprofile"
	"github.com/calyptia/cli/commands/tracesession"
	cfg "github.com/calyptia/cli/config"
)

func newCmdCreate(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		invitation.NewCmdSendInvitation(config),
		coreinstance.NewCmdCreateCoreInstance(config),
		coreinstance.NewCmdCreateCoreInstanceFile(config),
		coreinstance.NewCmdCreateCoreInstanceSecret(config),
		pipeline.NewCmdCreatePipeline(config),
		resourceprofile.NewCmdCreateResourceProfile(config),
		pipeline.NewCmdCreatePipelineFile(config),
		pipeline.NewCmdCreatePipelineLog(config),
		environment.NewCmdCreateEnvironment(config),
		tracesession.NewCmdCreateTraceSession(config),
		cnfg.NewCmdCreateConfigSection(config),
		ingestcheck.NewCmdCreateIngestCheck(config),
		fleet.NewCmdCreateFleet(config),
		fleet.NewCmdCreateFleetFile(config),
	)

	return cmd
}
