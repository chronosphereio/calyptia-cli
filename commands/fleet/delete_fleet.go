package fleet

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
)

func NewCmdDeleteFleet(cfg *config.Config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:               "fleet FLEET",
		Short:             "delete a fleet by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteFleets,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetKey := args[0]

			if !confirmed {
				cmd.Printf("Are yo sure you want to delete %q? (y/N)", fleetKey)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			_, err = cfg.Cloud.DeleteFleet(ctx, fleetID)
			if err != nil {
				return fmt.Errorf("could not delete pipeline: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")

	return cmd
}
