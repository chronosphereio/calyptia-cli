package agent

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calyptia/api/types"

	"github.com/chronosphereio/calyptia-cli/completer"
	cfg "github.com/chronosphereio/calyptia-cli/config"
	"github.com/chronosphereio/calyptia-cli/confirm"
)

func NewCmdDeleteAgent(config *cfg.Config) *cobra.Command {
	var confirmed bool
	var environment string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Delete a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			agentKey := args[0]
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			agentID, err := completer.LoadAgentID(agentKey, environmentID)
			if err != nil {
				return err
			}

			if !confirmed {
				cmd.Printf("Are you sure you want to delete agent with id %q? (y/N) ", agentID)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			err = config.Cloud.DeleteAgent(ctx, agentID)
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

	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)

	return cmd
}

func NewCmdDeleteAgents(config *cfg.Config) *cobra.Command {
	var inactive bool
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Delete many agents from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			aa, err := config.Cloud.Agents(ctx, config.ProjectID, types.AgentsParams{
				Last: cfg.Ptr(uint(0)),
			})
			if err != nil {
				return fmt.Errorf("could not prefetch agents to delete: %w", err)
			}

			if inactive {
				var onlyInactive []types.Agent
				for _, a := range aa.Items {
					inactive := a.LastMetricsAddedAt == nil || a.LastMetricsAddedAt.IsZero() || a.LastMetricsAddedAt.Before(time.Now().Add(time.Minute*-5))
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
				cmd.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (y/N) ", strings.Join(completer.AgentsKeys(aa.Items), "\n"))
				confirmed, err := confirm.Read(cmd.InOrStdin())
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

			err = config.Cloud.DeleteAgents(ctx, config.ProjectID, agentIDs...)
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
