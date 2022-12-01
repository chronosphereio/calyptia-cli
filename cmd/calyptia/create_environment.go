package main

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/api/types"
)

func newCmdCreateEnvironment(c *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environment ENVIRONMENT_NAME",
		Args:  cobra.ExactArgs(1),
		Short: "Create an environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			ctx := cmd.Context()
			environment := types.CreateEnvironment{Name: name}
			createEnvironment, err := c.cloud.CreateEnvironment(ctx, c.projectID, environment)
			if err != nil {
				return err
			}
			cmd.Printf("Created environment ID: %s Name: %s\n", createEnvironment.ID, name)
			return nil
		},
	}
	return cmd
}
