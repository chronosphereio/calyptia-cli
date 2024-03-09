package coreinstance

import (
	"fmt"

	"github.com/spf13/cobra"

	cfg "github.com/chronosphereio/calyptia-cli/config"
)

func NewCmdCreateCoreInstance(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "core_instance",
		Short:             "Setup a new core instance on a Kubernetes cluster.",
		PersistentPreRunE: checkForProjectToken(config),
	}
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
