package main

import (
	"github.com/spf13/cobra"
)

func newCmdDeleteIngestCheck(c *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest-check INGEST_CHECK_ID",
		Short: "Delete a specific ingest check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]
			err := c.cloud.DeleteIngestCheck(ctx, id)
			if err != nil {
				return err
			}
			cmd.Printf("Ingest check %s deleted successfully", id)
			return nil
		},
	}
	return cmd
}
