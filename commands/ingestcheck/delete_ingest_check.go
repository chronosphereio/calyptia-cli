package ingestcheck

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/config"
)

func NewCmdDeleteIngestCheck(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest_check INGEST_CHECK_ID",
		Short: "Delete a specific ingest check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]
			err := cfg.Cloud.DeleteIngestCheck(ctx, id)
			if err != nil {
				return err
			}
			cmd.Printf("Ingest check %s deleted successfully", id)
			return nil
		},
	}
	return cmd
}
