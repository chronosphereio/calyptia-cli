package cmd

import (
	cnfg "github.com/calyptia/cli/cmd/config"
	"github.com/calyptia/cli/cmd/coreinstance"
	"github.com/calyptia/cli/cmd/environment"
	"github.com/calyptia/cli/cmd/fleet"
	"github.com/calyptia/cli/cmd/ingestcheck"
	"github.com/calyptia/cli/cmd/pipeline"
	"github.com/calyptia/cli/cmd/resourceprofile"
	"github.com/calyptia/cli/cmd/tracesession"
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
)

func newCmdCreate(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		coreinstance.NewCmdCreateCoreInstance(config),
		pipeline.NewCmdCreatePipeline(config),
		resourceprofile.NewCmdCreateResourceProfile(config),
		pipeline.NewCmdCreatePipelineFile(config),
		environment.NewCmdCreateEnvironment(config),
		tracesession.NewCmdCreateTraceSession(config),
		cnfg.NewCmdCreateConfigSection(config),
		ingestcheck.NewCmdCreateIngestCheck(config),
		fleet.NewCmdCreateFleet(config),
	)

	return cmd
}
