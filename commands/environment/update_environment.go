package environment

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdateEnvironment(cfg *config.Config) *cobra.Command {
	var newName string
	cmd := &cobra.Command{
		Use:   "environment ENVIRONMENT_NAME",
		Args:  cobra.ExactArgs(1),
		Short: "Update an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if name == newName {
				return fmt.Errorf("environment name unchanged")
			}
			ctx := context.Background()
			environments, err := cfg.Cloud.Environments(ctx, cfg.ProjectID, cloudtypes.EnvironmentsParams{Name: &name})
			if err != nil {
				return err
			}
			if len(environments.Items) == 0 {
				return fmt.Errorf("environment not found")
			}
			environment := environments.Items[0]

			err = cfg.Cloud.UpdateEnvironment(ctx, environment.ID, cloudtypes.UpdateEnvironment{Name: &newName})
			if err != nil {
				return err
			}
			cmd.Printf("Updated environment ID: %s Name: %s\n", environment.ID, newName)
			return nil
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&newName, "name", "", "New environment name")
	return cmd
}
