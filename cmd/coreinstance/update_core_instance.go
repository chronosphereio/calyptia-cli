package coreinstance

import (
	"github.com/spf13/cobra"

	cfg "github.com/calyptia/cli/config"
)

func NewCmdUpdateCoreInstance(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Update a core instance on either Kubernetes, Amazon EC2 (TODO), or Google Compute Engine (TODO)",
	}
	cmd.AddCommand(NewCmdUpdateCoreInstanceK8s(config, nil))
	cmd.AddCommand(NewCmdUpdateCoreInstanceOperator(config, nil))
	cmd.AddCommand(NewCmdUpdateCoreInstanceOnAWS(config))
	cmd.AddCommand(NewCmdUpdateCoreInstanceOnGCP(config))
	return cmd
}
