package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdateAgent(cfg *config.Config) *cobra.Command {
	var newName string
	var fleetKey string
	var environment string

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Update a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			agentKey := args[0]
			agentID, err := cfg.Completer.LoadAgentID(ctx, agentKey)
			if err != nil {
				return err
			}

			fs := cmd.Flags()

			var in cloudtypes.UpdateAgent
			if fs.Changed("new-name") {
				in.Name = &newName
			}
			if fs.Changed("environment") {
				envID, err := cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
				in.EnvironmentID = &envID
			}
			if fs.Changed("fleet") {
				fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
				if err != nil {
					return err
				}

				in.FleetID = &fleetID
			}

			err = cfg.Cloud.UpdateAgent(ctx, agentID, in)
			if err != nil {
				return fmt.Errorf("could not update agent: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&newName, "new-name", "", "New agent name")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVar(&fleetKey, "fleet", "", "Attach this agent to the given fleet")

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("fleet", cfg.Completer.CompleteFleets)

	return cmd
}
