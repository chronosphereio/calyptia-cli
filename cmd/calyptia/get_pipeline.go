package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetPipelines(config *config) *cobra.Command {
	var format string
	var aggregatorKey string
	var last uint64
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Display latest pipelines from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorID := aggregatorKey
			if !validUUID(aggregatorID) {
				aa, err := config.fetchAllAggregators()
				if err != nil {
					return err
				}

				a, ok := findAggregatorByName(aa, aggregatorKey)
				if !ok {
					return fmt.Errorf("could not find aggregator %q", aggregatorKey)
				}

				aggregatorID = a.ID
			}
			pp, err := config.cloud.AggregatorPipelines(config.ctx, aggregatorID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				// TODO: use pipeline name.
				tw.AppendHeader(table.Row{"ID", "Replica size", "Status", "Created at"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, p := range pp {
					tw.AppendRow(table.Row{p.ID, p.ReplicasCount, p.Status.Status, p.CreatedAt})
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
	fs.StringVar(&aggregatorKey, "aggregator", "", "Parent aggregator ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("aggregator", config.completeAggregators)

	_ = cmd.MarkFlagRequired("aggregator-id") // TODO: use default aggregator ID from config cmd.

	return cmd
}
