package main

import (
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/spf13/cobra"
)

func newCmdCreate(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create core instances, pipelines, etc.",
	}

	cmd.AddCommand(
		newCmdCreateCoreInstance(config),
		newCmdCreatePipeline(config),
		newCmdCreateResourceProfile(config),
		newCmdCreatePipelineFile(config),
		newCmdCreateEnvironment(config),
		newCmdCreateTraceSession(config),
		newCmdCreateConfigSection(config),
		newCmdCreateIngestCheck(config),
		newCmdCreateFleet(config),
	)

	return cmd
}
