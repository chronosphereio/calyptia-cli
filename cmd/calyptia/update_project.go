package main

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
)

func newCmdUpdateProject(config *cfg.Config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Update the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
				return nil
			}

			err := config.Cloud.UpdateProject(config.Ctx, config.ProjectID, cloud.UpdateProject{
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
