package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetProjects(config *config) *cobra.Command {
	var format string
	var last uint64
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Display latest projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			pp, err := config.cloud.Projects(config.ctx, last)
			if err != nil {
				return fmt.Errorf("could not fetch your projects: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"ID", "Name", "Created at"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, p := range pp {
					tw.AppendRow(table.Row{p.ID, p.Name, p.CreatedAt})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your projects: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&format, "output-format", "f", "table", "Output format. Allowed: table, json")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` projects. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
