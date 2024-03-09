package coreinstance

import (
	"github.com/spf13/cobra"

	cfg "github.com/chronosphereio/calyptia-cli/config"
)

func NewCmdUpdateCoreInstance(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Update a core instance on a Kubernetes cluster.",
	}
	cmd.AddCommand(NewCmdUpdateCoreInstanceOperator(config, nil))
	return cmd
}
