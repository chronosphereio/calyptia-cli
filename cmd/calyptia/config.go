package main

import (
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/spf13/cobra"
)

func newCmdConfig(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure Calyptia CLI",
	}

	cmd.AddCommand(
		newCmdConfigSetToken(config),
		newCmdConfigCurrentToken(config),
		newCmdConfigUnsetToken(config),
		newCmdConfigSetURL(config),
		newCmdConfigCurrentURL(config),
		newCmdConfigUnsetURL(config),
	)

	return cmd
}
