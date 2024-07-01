package environment

import (
	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdCreateEnvironment(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environment ENVIRONMENT_NAME",
		Args:  cobra.ExactArgs(1),
		Short: "Create an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := args[0]
			environment := cloudtypes.CreateEnvironment{Name: name}
			createEnvironment, err := cfg.Cloud.CreateEnvironment(ctx, cfg.ProjectID, environment)
			if err != nil {
				return err
			}
			cmd.Printf("Created environment ID: %s Name: %s\n", createEnvironment.ID, name)
			return nil
		},
	}
	return cmd
}
