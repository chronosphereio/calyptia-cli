package ingestcheck

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calyptia/cli/config"
)

func NewCmdGetIngestCheckLogs(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingest_check_logs INGEST_CHECK_ID",
		Short: "Get a specific ingest check logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]
			check, err := cfg.Cloud.IngestCheck(ctx, id)
			if err != nil {
				return err
			}
			fmt.Println(string(check.Logs))
			return nil
		},
	}
	return cmd
}
