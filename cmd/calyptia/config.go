package main

import "github.com/spf13/cobra"

func newCmdConfig(config *config) *cobra.Command {
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
