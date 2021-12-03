package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/calyptia/cloud"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdCreateAggregator(config *config) *cobra.Command {
	var projectKey string
	var name string
	var format string
	cmd := &cobra.Command{
		Use:   "aggregator",
		Short: "Create a new aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectKey == "" {
				return errors.New("project required")
			}

			projectID := projectKey
			{
				pp, err := config.cloud.Projects(config.ctx, 0)
				if err != nil {
					return err
				}

				p, ok := findProjectByName(pp, projectKey)
				if !ok && !validUUID(projectID) {
					return fmt.Errorf("could not find project %q", projectKey)
				}

				if ok {
					projectID = p.ID
				}
			}

			a, err := config.cloud.CreateAggregator(config.ctx, cloud.CreateAggregatorPayload{
				Name: name,
			}, cloud.CreateAggregatorWithProjectID(projectID))
			if err != nil {
				return fmt.Errorf("could not create aggregator: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"Name", "Age"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}
				tw.AppendRow(table.Row{a.Name, fmtAgo(a.CreatedAt)})
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(a)
				if err != nil {
					return fmt.Errorf("could not json encode your new aggregator: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&projectKey, "project", config.defaultProject, "Parent project ID or name")
	fs.StringVar(&name, "name", "", "Aggregator name; leave it empty to generate a random name")
	fs.StringVarP(&format, "output-format", "f", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("project", config.completeProjects)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
