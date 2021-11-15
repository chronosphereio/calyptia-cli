package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetPipelinePorts(config *config) *cobra.Command {
	var format string
	var pipelineID string
	var last uint64
	cmd := &cobra.Command{
		Use:   "pipeline_ports",
		Short: "Display latest ports from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			pp, err := config.cloud.PipelinePorts(config.ctx, pipelineID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline ports: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"ID", "Protocol", "Frontend port", "Backend port", "Endpoint", "Created at"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, p := range pp {
					tw.AppendRow(table.Row{p.ID, p.Protocol, p.FrontendPort, p.BackendPort, p.Endpoint, p.CreatedAt})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline ports: %w", err)
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
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipeline ports. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline-id", config.completePipelineIDs)

	_ = cmd.MarkFlagRequired("pipeline-id") // TODO: use default pipeline ID from config cmd.

	return cmd
}
