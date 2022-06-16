package main

import (
	"github.com/spf13/cobra"
)

func newCmdCreateAggregator(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "core_instance",
		Short: "Setup a new core instance on either Kubernetes, Amazon EC2 (TODO), or Google Compute Engine (TODO)",
	}
	cmd.AddCommand(newCmdCreateAggregatorOnK8s(config, nil))
	cmd.AddCommand(newCmdCreateAggregatorOnAWS(config))
	cmd.AddCommand(newCmdCreateAggregatorOnGCP(config))
	return cmd
}
