package coreinstance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calyptia/cli/config"
)

func NewCmdCreateCoreInstance(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "core_instance",
		Short:             "Setup a new core instance on a Kubernetes cluster.",
		PersistentPreRunE: checkForProjectToken(cfg),
	}
	cmd.AddCommand(newCmdCreateCoreInstanceOperator(cfg, nil))
	return cmd
}

func checkForProjectToken(cfg *config.Config) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if cfg.ProjectToken == "" {
			return fmt.Errorf("project token is required to realize this action.\nPlease set it with `calyptia config set_token <token>`")
		}
		return nil
	}
}
