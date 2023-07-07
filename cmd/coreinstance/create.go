package coreinstance

import (
	"fmt"
	"github.com/spf13/cobra"

	cfg "github.com/calyptia/cli/config"
)

func NewCmdCreateCoreInstance(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "core_instance",
		Short:             "Setup a new core instance on either Kubernetes, Amazon EC2, or Google Compute Engine",
		PersistentPreRunE: checkForProjectToken(config),
	}
	cmd.AddCommand(newCmdCreateCoreInstanceOnK8s(config, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOnAWS(config, nil, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOnGCP(config, nil))
	cmd.AddCommand(newCmdCreateCoreInstanceOperator(config, nil))
	return cmd
}

func checkForProjectToken(config *cfg.Config) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if config.ProjectToken == "" {
			return fmt.Errorf("project token is required to realize this action.\nPlease set it with `calyptia config set_token <token>`")
		}
		return nil
	}
}
