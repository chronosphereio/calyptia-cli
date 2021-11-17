package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newCmdDeleteAgent(config *config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:               "agent AGENT",
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

				answer = strings.TrimSpace(strings.ToLower(answer))
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

func newCmdDeleteAgents(config *config) *cobra.Command {
	var projectKey string
	var inactive bool
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Delete many agents from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := projectKey
			if !validUUID(projectID) {
				pp, err := config.cloud.Projects(config.ctx, 0)
				if err != nil {
					return err
				}

				a, ok := findProjectByName(pp, projectKey)
				if !ok {
					return nil
				}

				projectID = a.ID
			}

			aa, err := config.cloud.Agents(config.ctx, projectID, 0)
			if err != nil {
				return fmt.Errorf("could not prefetch agents to delete: %w", err)
			}

			if inactive {
				var onlyInactive []cloud.Agent
				for _, a := range aa {
					inactive := a.LastMetricsAddedAt.IsZero() || a.LastMetricsAddedAt.Before(time.Now().Add(time.Minute*-5))
					if inactive {
						onlyInactive = append(onlyInactive, a)
					}
				}
				aa = onlyInactive
			}

			if len(aa) == 0 {
				fmt.Println("Nothing to delete")
				return nil
			}

			if !confirmed {
				fmt.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (yes/N) ", strings.Join(agentsKeys(aa), "\n"))
				var answer string
				_, err := fmt.Scanln(&answer)
				if err != nil && err.Error() == "unexpected newline" {
					err = nil
				}

				if err != nil {
					return fmt.Errorf("could not to read answer: %v", err)
				}

				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "yes" {
					return nil
				}
			}

			g, gctx := errgroup.WithContext(config.ctx)
			for _, a := range aa {
				a := a
				g.Go(func() error {
					err := config.cloud.DeleteAgent(gctx, a.ID)
					if err != nil {
						return fmt.Errorf("could not delete agent %q: %w", a.ID, err)
					}

					return nil
				})
			}
			if err := g.Wait(); err != nil {
				return err
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&projectKey, "project", "", "Delete agents from this project ID or name")
	fs.BoolVar(&inactive, "inactive", true, "Delete inactive agents only")
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	_ = cmd.RegisterFlagCompletionFunc("project", config.completeProjects)

	_ = cmd.MarkFlagRequired("project") // TODO: use default project ID from config cmd.

	return cmd
}
