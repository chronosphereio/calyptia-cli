package main

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

func newCmdDeleteCoreInstance(config *config, testClientSet kubernetes.Interface) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "core_instance",
		Aliases: []string{"instance", "aggregator"},
		Short:   "Delete a core instance from either Kubernetes, Amazon EC2 (TODO), or Google Compute Engine (TODO)",
	}
	cmd.AddCommand(newCmdDeleteCoreInstanceK8s(config, nil))
	cmd.AddCommand(newCmdDeleteCoreInstanceOnAWS(config, nil))
	cmd.AddCommand(newCmdDeleteCoreInstanceOnGCP(config))
	return cmd
}
