package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/calyptia/api/types"
	cloud "github.com/calyptia/api/types"
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
			agentKey := args[0]
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			if !confirmed {
				fmt.Printf("Are you sure you want to delete %q? (y/N) ", agentKey)
				confirmed, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			agentID, err := config.loadAgentID(agentKey, environmentID)
			if err != nil {
				return err
			}

			err = config.cloud.DeleteAgent(config.ctx, agentID)
			if err != nil {
				return fmt.Errorf("could not delete agent: %w", err)
			}

			cmd.Printf("Agent with id %q deleted\n", agentID)

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
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
			if !confirmed {
				fmt.Print("Are you sure you want to delete all agents? (y/N) ")
				confirmed, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			aa, err := config.cloud.Agents(config.ctx, config.projectID, cloud.AgentsParams{
				Last: ptr(uint(200)),
			})
			if err != nil {
				return fmt.Errorf("could not prefetch agents to delete: %w", err)
			}

			if inactive {
				var onlyInactive []cloud.Agent
				for _, a := range aa.Items {
					inactive := a.LastMetricsAddedAt.IsZero() || a.LastMetricsAddedAt.Before(time.Now().Add(time.Minute*-5))
					if inactive {
						onlyInactive = append(onlyInactive, a)
					}
				}
				aa.Items = onlyInactive
			}

			if len(aa.Items) == 0 {
				cmd.Println("No agents left to delete")
				return nil
			}

			cmd.Printf("About to delete %d agents\n", len(aa.Items))

			g := sync.WaitGroup{}

			var count uint
			for _, a := range aa.Items {
				g.Add(1)
				go func(a types.Agent) {
					defer g.Done()

					err := config.cloud.DeleteAgent(config.ctx, a.ID)
					if err != nil {
						cmd.PrintErrf("could not delete agent %q: %v\n", a.ID, err)
						return
					}

					count++
				}(a)
			}

			g.Wait()

			cmd.Printf("Deleted %d agents\n", count)

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&inactive, "inactive", true, "Delete inactive agents only")
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	return cmd
}
