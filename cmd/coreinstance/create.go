package coreinstance

import (
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
)

func NewCmdCreateCoreInstance(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Setup a new core instance on either Kubernetes, Amazon EC2, or Google Compute Engine",
	}
	cmd.AddCommand(newCmdCreateCoreInstanceOnK8s(config, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOnAWS(config, nil, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOnGCP(config, nil))
	return cmd
}
