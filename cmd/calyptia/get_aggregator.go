package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetAggregators(config *config) *cobra.Command {
	var format string
	var projectID string
	var last uint64
	cmd := &cobra.Command{
		Use:   "aggregators",
		Short: "Display latest aggregators from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			aa, err := config.cloud.Aggregators(config.ctx, projectID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your aggregators: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"ID", "Name", "Created at"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, a := range aa {
					tw.AppendRow(table.Row{a.ID, a.Name, a.CreatedAt})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(aa)
				if err != nil {
					return fmt.Errorf("could not json encode your aggregators: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&projectID, "project-id", "", "Parent project ID")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` aggregators. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("project-id", config.completeProjectIDs)

	_ = cmd.MarkFlagRequired("project-id") // TODO: use default project ID from config cmd.

	return cmd
}
