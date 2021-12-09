package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdCreateAggregator(config *config) *cobra.Command {
	var projectKey string
	var name string
	var addHealthCheckPipeline bool
	var healthCheckPipelinePort uint
	var format string
	cmd := &cobra.Command{
		Use:   "aggregator",
		Short: "Create a new aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectKey == "" {
				return errors.New("project required")
			}

			projectID, err := config.loadProjectID(projectKey)
			if err != nil {
				return err
			}

			a, err := config.cloud.CreateAggregator(config.ctx, cloud.AddAggregatorPayload{
				Name:                    name,
				AddHealthCheckPipeline:  addHealthCheckPipeline,
				HealthCheckPipelinePort: healthCheckPipelinePort,
			}, cloud.CreateAggregatorWithProjectID(projectID))
			if err != nil {
				return fmt.Errorf("could not create aggregator: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "NAME\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", a.Name, fmtAgo(a.CreatedAt))
				tw.Flush()
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
	fs.BoolVar(&addHealthCheckPipeline, "healthcheck", true, "Add a health check pipeline by default with the aggregator")
	fs.UintVar(&healthCheckPipelinePort, "healthcheck-port", 2020, "Health check pipeline port if a health check pipeline is added")
	fs.StringVarP(&format, "output-format", "f", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("project", config.completeProjects)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
