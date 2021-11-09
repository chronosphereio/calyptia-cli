package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetPipelineConfigHistory(config *config) *cobra.Command {
	var format string
	var pipelineID string
	var last uint64
	cmd := &cobra.Command{
		Use:   "pipeline_config_history",
		Short: "Display latest config history from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, err := config.cloud.PipelineConfigHistory(config.ctx, pipelineID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline config history: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"ID", "Created at"})
				tw.SetStyle(table.StyleRounded)
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, c := range cc {
					tw.AppendRow(table.Row{c.ID, c.CreatedAt})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(cc)
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
	fs.StringVar(&pipelineID, "pipeline-id", "", "Parent pipeline ID")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipeline config history entries. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	// _ = cmd.RegisterFlagCompletionFunc("pipeline-id", nil) // TODO: complete pipelineID.

	_ = cmd.MarkFlagRequired("pipeline-id") // TODO: use default pipeline ID from config cmd.

	return cmd
}
