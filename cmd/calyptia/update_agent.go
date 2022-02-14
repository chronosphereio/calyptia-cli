package main

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdUpdateAgent(config *config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Update a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
				return nil
			}

			agentKey := args[0]
			// We can only update the agent name. Early return if its the same.
			if agentKey == newName {
				return nil
			}

			agentID, err := config.loadAgentID(agentKey)
			if err != nil {
				return err
			}

			err = config.cloud.UpdateAgent(config.ctx, agentID, cloud.UpdateAgent{
				Name: &newName,
			})
			if err != nil {
				return fmt.Errorf("could not update agent: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&newName, "new-name", "", "New agent name")

	_ = cmd.MarkFlagRequired("new-name")

	return cmd
}
