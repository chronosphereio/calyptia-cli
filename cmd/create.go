package cmd

import (
	"github.com/spf13/cobra"

	cnfg "github.com/calyptia/cli/cmd/config"
	"github.com/calyptia/cli/cmd/coreinstance"
	"github.com/calyptia/cli/cmd/environment"
	"github.com/calyptia/cli/cmd/fleet"
	"github.com/calyptia/cli/cmd/ingestcheck"
	"github.com/calyptia/cli/cmd/invitation"
	"github.com/calyptia/cli/cmd/pipeline"
	"github.com/calyptia/cli/cmd/resourceprofile"
	"github.com/calyptia/cli/cmd/tracesession"
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
		environment.NewCmdCreateEnvironment(config),
		tracesession.NewCmdCreateTraceSession(config),
		cnfg.NewCmdCreateConfigSection(config),
		ingestcheck.NewCmdCreateIngestCheck(config),
		fleet.NewCmdCreateFleet(config),
		fleet.NewCmdCreateFleetFile(config),
	)

	return cmd
}
