package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetPipelineStatusHistory(config *config) *cobra.Command {
	var pipelineKey string
	var last uint
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "pipeline_status_history",
		Short: "Display latest status history from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ss, err := config.cloud.PipelineStatusHistory(config.ctx, pipelineID, cloud.PipelineStatusHistoryParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline status history: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "STATUS\tCONFIG-ID\tAGE")
				for _, s := range ss.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", s.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", s.Status, s.Config.ID, fmtTime(s.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline status history: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline status history entries. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
