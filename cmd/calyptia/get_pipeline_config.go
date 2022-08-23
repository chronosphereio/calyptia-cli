package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetPipelineConfigHistory(config *config) *cobra.Command {
	var format string
	var pipelineKey string
	var last uint
	cmd := &cobra.Command{
		Use:   "pipeline_config_history",
		Short: "Display latest config history from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			cc, err := config.cloud.PipelineConfigHistory(config.ctx, pipelineID, cloud.PipelineConfigHistoryParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline config history: %w", err)
			}

			switch format {
			case "table":
				renderPipelineConfigHistory(cmd.OutOrStdout(), cc.Items)
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(cc.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline config history: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline config history entries. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineConfigHistory(w io.Writer, cc []cloud.PipelineConfig) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tAGE")
	for _, c := range cc {
		fmt.Fprintf(tw, "%s\t%s\n", c.ID, fmtAgo(c.CreatedAt))
	}
	tw.Flush()
}
