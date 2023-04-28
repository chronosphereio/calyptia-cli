package fleet

import (
	"fmt"
	"strings"

	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
)

func NewCmdDeleteFleet(config *config.Config) *cobra.Command {
	var confirmed bool
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "fleet FLEET",
		Short:             "delete a fleet by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteFleets,
		RunE: func(cmd *cobra.Command, args []string) error {
			fleetKey := args[0]
			if !confirmed {
				cmd.Printf("Are yo sure you want to delete %q? (y/N)", fleetKey)
				var answer string
				_, err := fmt.Scanln(&answer)
				if err != nil && err.Error() == "unexpected  newline" {
					err = nil
				}

				if err != nil {
					return fmt.Errorf("could not read answer: %v", err)
				}

				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					return nil
				}
			}

			fleetID, err := completer.LoadFleetID(fleetKey)
			if err != nil {
				return err
			}

			_, err = config.Cloud.DeleteFleet(config.Ctx, fleetID)
			if err != nil {
				return fmt.Errorf("could not delete pipeline: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	return cmd
}
