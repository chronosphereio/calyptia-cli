package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCmdTopAgent(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "agent id",
		Short: "Display metrics from an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentID := args[0]
			fmt.Println("TODO: show agent metrics:", agentID)
			return nil
		},
	}
}
