package main

import (
	"fmt"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
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
			agentID := agentKey
			{
				// We can only update the agent name. Early return if its the same.
				if agentKey == newName {
					return nil
				}

				aa, err := config.fetchAllAgents()
				if err != nil {
					return err
				}

				a, ok := findAgentByName(aa, agentKey)
				if !ok && !validUUID(agentID) {
					return fmt.Errorf("could not find agent %q", agentKey)
				}

				if ok {
					agentID = a.ID
				}
			}

			err := config.cloud.UpdateAgent(config.ctx, agentID, cloud.UpdateAgentOpts{
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
