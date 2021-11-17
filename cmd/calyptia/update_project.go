package main

import (
	"fmt"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdUpdateProject(config *config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:               "project PROJECT",
		Short:             "Update a single project by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
				return nil
			}

			projectKey := args[0]
			projectID := projectKey
			if !validUUID(projectID) {
				if projectKey == newName {
					return nil
				}

				pp, err := config.cloud.Projects(config.ctx, 0)
				if err != nil {
					return err
				}

				a, ok := findProjectByName(pp, projectKey)
				if !ok {
					return fmt.Errorf("could not find project %q", projectKey)
				}

				projectID = a.ID
			}

			err := config.cloud.UpdateProject(config.ctx, projectID, cloud.UpdateProjectOpts{
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
