package environment

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdCreateEnvironment(c *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environment ENVIRONMENT_NAME",
		Args:  cobra.ExactArgs(1),
		Short: "Create an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := context.Background()
			environment := types.CreateEnvironment{Name: name}
			createEnvironment, err := c.Cloud.CreateEnvironment(ctx, c.ProjectID, environment)
			if err != nil {
				return err
			}
			cmd.Printf("Created environment ID: %s Name: %s\n", createEnvironment.ID, name)
			return nil
		},
	}
	return cmd
}
