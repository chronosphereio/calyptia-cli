package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calyptia/api/types"
)

func newCmdDeleteAgent(config *config) *cobra.Command {
	var confirmed bool
	var environment string

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Delete a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			agentKey := args[0]
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			agentID, err := config.loadAgentID(agentKey, environmentID)
			if err != nil {
				return err
			}

			if !confirmed {
				cmd.Printf("Are you sure you want to delete agent with id %q? (y/N) ", agentID)
				confirmed, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			err = config.cloud.DeleteAgent(ctx, agentID)
			if err != nil {
				return fmt.Errorf("could not delete agent: %w", err)
			}

			cmd.Printf("Successully deleted agent with id %q\n", agentID)

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)

	return cmd
}

func newCmdDeleteAgents(config *config) *cobra.Command {
	var inactive bool
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Delete many agents from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			aa, err := config.cloud.Agents(ctx, config.projectID, types.AgentsParams{
				Last: ptr(uint(0)),
			})
			if err != nil {
				return fmt.Errorf("could not prefetch agents to delete: %w", err)
			}

			if inactive {
				var onlyInactive []types.Agent
				for _, a := range aa.Items {
					inactive := a.LastMetricsAddedAt.IsZero() || a.LastMetricsAddedAt.Before(time.Now().Add(time.Minute*-5))
					if inactive {
						onlyInactive = append(onlyInactive, a)
					}
				}
				aa.Items = onlyInactive
			}

			if len(aa.Items) == 0 {
				cmd.Println("No agents to delete")
				return nil
			}

			if !confirmed {
				cmd.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (y/N) ", strings.Join(agentsKeys(aa.Items), "\n"))
				confirmed, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			agentIDs := make([]string, len(aa.Items))
			for i, a := range aa.Items {
				agentIDs[i] = a.ID
			}

			err = config.cloud.DeleteAgents(ctx, config.projectID, agentIDs...)
			if err != nil {
				return fmt.Errorf("delete agents: %w", err)
			}

			cmd.Printf("Successfully deleted %d agents\n", len(agentIDs))

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVar(&inactive, "inactive", true, "Delete inactive agents only")
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")

	return cmd
}
