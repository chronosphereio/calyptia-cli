package main

import (
	"fmt"

	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
)

func newCmdUpdateProject(config *config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Update the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
				return nil
			}

			err := config.cloud.UpdateProject(config.ctx, config.projectID, cloud.UpdateProject{
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
