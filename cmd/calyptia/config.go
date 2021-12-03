package main

import "github.com/spf13/cobra"

func newCmdConfig(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure Calyptia CLI",
	}

	cmd.AddCommand(
		newCmdConfigSetProject(config),
		newCmdConfigCurrentProject(config),
		newCmdConfigUnsetProject(config),
	)

	return cmd
}
