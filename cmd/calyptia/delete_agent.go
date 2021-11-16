package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCmdDeleteAgent(config *config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:               "agent key",
		Short:             "Delete a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentKey := args[0]
			if !confirmed {
				fmt.Printf("Are you sure you want to delete %q? (y/N) ", agentKey)
				var answer string
				_, err := fmt.Scanln(&answer)
				if err != nil && err.Error() == "unexpected newline" {
					err = nil
				}

				if err != nil {
					return fmt.Errorf("could not to read answer: %v", err)
				}

				answer = strings.ToLower(answer)
				if answer != "y" && answer != "yes" {
					return nil
				}
			}

			agentID := agentKey
			if !validUUID(agentID) {
				aa, err := config.fetchAllAgents()
				if err != nil {
					return err
				}

				a, ok := findAgentByName(aa, agentKey)
				if !ok {
					return nil
				}

				agentID = a.ID
			}

			err := config.cloud.DeleteAgent(config.ctx, agentID)
			if err != nil {
				return fmt.Errorf("could not delete agent: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	return cmd
}
