package coreinstance

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/config"
)

func NewCmdUpdateCoreInstance(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Update a core instance on a Kubernetes cluster.",
	}
	cmd.AddCommand(NewCmdUpdateCoreInstanceOperator(cfg, nil))
	return cmd
}
