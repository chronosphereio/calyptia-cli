package main

import (
	"github.com/calyptia/cli/cmd/calyptia/utils"
	"github.com/spf13/cobra"
)

func newCmdUpdateCoreInstance(config *utils.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Update a core instance on either Kubernetes, Amazon EC2 (TODO), or Google Compute Engine (TODO)",
	}
	cmd.AddCommand(newCmdUpdateCoreInstanceK8s(config, nil))
	cmd.AddCommand(newCmdUpdateCoreInstanceOnAWS(config))
	cmd.AddCommand(newCmdUpdateCoreInstanceOnGCP(config))
	return cmd
}
