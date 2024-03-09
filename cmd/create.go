package cmd

import (
	"github.com/spf13/cobra"

	cnfg "github.com/chronosphereio/calyptia-cli/cmd/config"
	"github.com/chronosphereio/calyptia-cli/cmd/coreinstance"
	"github.com/chronosphereio/calyptia-cli/cmd/environment"
	"github.com/chronosphereio/calyptia-cli/cmd/fleet"
	"github.com/chronosphereio/calyptia-cli/cmd/ingestcheck"
	"github.com/chronosphereio/calyptia-cli/cmd/invitation"
	"github.com/chronosphereio/calyptia-cli/cmd/pipeline"
	"github.com/chronosphereio/calyptia-cli/cmd/resourceprofile"
	"github.com/chronosphereio/calyptia-cli/cmd/tracesession"
	cfg "github.com/chronosphereio/calyptia-cli/config"
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
