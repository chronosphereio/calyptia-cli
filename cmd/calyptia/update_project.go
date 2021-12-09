package main

import (
	"errors"
	"fmt"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdUpdateProject(config *config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:               "project [PROJECT]",
		Short:             "Update a single project by ID or name",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: config.completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
				return nil
			}

			projectKey := config.defaultProject
			if len(args) > 0 {
				projectKey = args[0]
			}
			if projectKey == "" {
				return errors.New("project required")
			}

			projectID, err := config.loadProjectID(projectKey)
			if err != nil {
				return err
			}

			err = config.cloud.UpdateProject(config.ctx, projectID, cloud.UpdateProjectOpts{
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
