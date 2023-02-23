package environment

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
)

func NewCmdDeleteEnvironment(c *cfg.Config) *cobra.Command {
	var confirmDelete bool
	cmd := &cobra.Command{
		Use:   "environment ENVIRONMENT_NAME",
		Args:  cobra.ExactArgs(1),
		Short: "Delete an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()
			environments, err := c.Cloud.Environments(ctx, c.ProjectID, types.EnvironmentsParams{Name: &name})
			if err != nil {
				return err
			}
			if len(environments.Items) == 0 {
				return fmt.Errorf("environment not found")
			}
			environment := environments.Items[0]
			if !confirmDelete {
				cmd.Print("This will remove ALL your agents, core_instances. Do you confirm? [y/N] ")
				confirmDelete, err = confirm.ReadConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmDelete {
					cmd.Println("Aborted")
					return nil
				}
			}

			err = c.Cloud.DeleteEnvironment(ctx, environment.ID)
			if err != nil {
				return err
			}
			cmd.Printf("Deleted environment ID: %s Name: %s\n", environment.ID, environment.Name)
			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVar(&confirmDelete, "yes", isNonInteractive, "Confirm deletion")

	return cmd
}
