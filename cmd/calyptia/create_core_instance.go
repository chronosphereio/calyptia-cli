package main

import (
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/spf13/cobra"
)

func newCmdCreateCoreInstance(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Setup a new core instance on either Kubernetes, Amazon EC2, or Google Compute Engine",
	}
	cmd.AddCommand(newCmdCreateCoreInstanceOnK8s(config, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOnAWS(config, nil, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOnGCP(config, nil))
	return cmd
}
