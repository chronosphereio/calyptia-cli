package coreinstance

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/pointer"
)

func NewCmdDeleteCoreInstance(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "core_instance",
		Aliases: []string{"instance", "core_instance"},
		Short:   "Delete a core instance from a Kubernetes cluster.",
	}
	cmd.AddCommand(
		NewCmdDeleteCoreInstanceOperator(cfg, nil),
	)
	return cmd
}

func NewCmdDeleteCoreInstances(cfg *config.Config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "core_instances",
		Short: "Delete many core instances from project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			aa, err := cfg.Cloud.CoreInstances(ctx, cfg.ProjectID, cloudtypes.CoreInstancesParams{
				Last: pointer.From(uint(0)),
			})
			if err != nil {
				return fmt.Errorf("could not prefetch core instances to delete: %w", err)
			}

			if len(aa.Items) == 0 {
				cmd.Println("No core instances to delete")
				return nil
			}

			if !confirmed {
				cmd.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (y/N) ", strings.Join(completer.CoreInstanceKeys(aa.Items), "\n"))
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			coreInstanceIDs := make([]string, len(aa.Items))
			for i, a := range aa.Items {
				coreInstanceIDs[i] = a.ID
			}

			err = cfg.Cloud.DeleteCoreInstances(ctx, cfg.ProjectID, coreInstanceIDs...)
			if err != nil {
				return fmt.Errorf("delete core instances: %w", err)
			}

			cmd.Printf("Successfully deleted %d core instances\n", len(coreInstanceIDs))

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")

	return cmd
}
