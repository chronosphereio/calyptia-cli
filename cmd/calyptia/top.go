package main

import "github.com/spf13/cobra"

func newCmdTop(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Display metrics",
	}

	cmd.AddCommand(
		newCmdTopProject(config),
		newCmdTopAgent(config),
	)

	return cmd
}
