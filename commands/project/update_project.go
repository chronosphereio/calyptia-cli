package project

import (
	"fmt"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdateProject(cfg *config.Config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Update the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if newName == "" {
				return nil
			}

			err := cfg.Cloud.UpdateProject(ctx, cfg.ProjectID, cloudtypes.UpdateProject{
				Name: &newName,
			})
			if err != nil {
				return fmt.Errorf("could not update project: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&newName, "new-name", "", "New project name")

	_ = cmd.MarkFlagRequired("new-name")

	return cmd
}
