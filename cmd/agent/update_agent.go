package agent

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdUpdateAgent(config *cfg.Config) *cobra.Command {
	var newName string
	var fleetKey string
	var environment string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Update a single agent by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentKey := args[0]

			agentID, err := completer.LoadAgentID(agentKey, "")
			if err != nil {
				return err
			}

			fs := cmd.Flags()

			var in cloud.UpdateAgent
			if fs.Changed("new-name") {
				in.Name = &newName
			}
			if fs.Changed("environment") {
				envID, err := completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
				in.EnvironmentID = &envID
			}
			if fs.Changed("fleet") {
				fleetID, err := completer.LoadFleetID(fleetKey)
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

	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("fleet", completer.CompleteFleets)

	return cmd
}
