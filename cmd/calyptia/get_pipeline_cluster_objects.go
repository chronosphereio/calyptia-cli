package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetPipelineClusterObjects(config *config) *cobra.Command {
	var pipelineKey string
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_objects",
		Short: "Get pipeline cluster objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			co, err := config.cloud.PipelineClusterObjects(config.ctx, pipelineID, cloud.PipelineClusterObjectsParams{
				Last: &last,
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, co.Items)
			}

			switch outputFormat {
			case "table":
				{
					tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
					if showIDs {
						fmt.Fprintf(tw, "ID\t")
					}
					fmt.Fprintln(tw, "NAME\tKIND\tCREATED AT")
					for _, c := range co.Items {
						if showIDs {
							fmt.Fprintf(tw, "%s\t", c.ID)
						}
						fmt.Fprintf(tw, "%s\t%s\t%s\n", c.Name, string(c.Kind), fmtTime(c.CreatedAt))
					}
					tw.Flush()
				}
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(co.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(co.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Pipeline to list cluster objects for")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` cluster objects. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}