package main

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/calyptia/utils"
)

func newCmdUpdateAgent(config *utils.Config) *cobra.Command {
	var newName string
	var fleetKey string
	var environment string

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Update a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.CompleteAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentKey := args[0]

			agentID, err := config.LoadAgentID(agentKey, "")
			if err != nil {
				return err
			}

			fs := cmd.Flags()

			var in cloud.UpdateAgent
			if fs.Changed("new-name") {
				in.Name = &newName
			}
			if fs.Changed("environment") {
				envID, err := config.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
				in.EnvironmentID = &envID
			}
			if fs.Changed("fleet") {
				fleetID, err := config.LoadFleetID(fleetKey)
				if err != nil {
					return err
				}

				in.FleetID = &fleetID
			}

			err = config.Cloud.UpdateAgent(config.Ctx, agentID, in)
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

	_ = cmd.RegisterFlagCompletionFunc("environment", config.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("fleet", config.CompleteFleets)

	return cmd
}
