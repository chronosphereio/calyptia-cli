package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetAggregatorPipelines(config *config) *cobra.Command {
	var format string
	var aggregatorID string
	var last uint64
	cmd := &cobra.Command{
		Use:   "aggregator_pipelines",
		Short: "Display latest pipelines from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			pp, err := config.cloud.AggregatorPipelines(config.ctx, aggregatorID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"ID", "Replica size", "Status", "Created at"})
				tw.SetStyle(table.StyleRounded)
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, p := range pp {
					tw.AppendRow(table.Row{p.ID, p.ReplicasCount, p.Status, p.CreatedAt})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your pipelines: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&aggregatorID, "aggregator-id", "", "Parent aggregator ID")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")

	cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	// cmd.RegisterFlagCompletionFunc("aggregator-id", nil) // TODO: complete aggregatorID.

	cmd.MarkFlagRequired("aggregator-id") // TODO: use default aggregator ID from config cmd.

	return cmd
}
