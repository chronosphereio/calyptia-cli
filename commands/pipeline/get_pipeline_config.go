package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetPipelineConfigHistory(config *cfg.Config) *cobra.Command {
	var outputFormat, goTemplate string
	var pipelineKey string
	var last uint

	cmd := &cobra.Command{
		Use:   "pipeline_config_history",
		Short: "Display latest config history from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := config.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			cc, err := config.Cloud.PipelineConfigHistory(ctx, pipelineID, types.PipelineConfigHistoryParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline config history: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, cc.Items)
			}

			switch outputFormat {
			case "table":
				renderPipelineConfigHistory(cmd.OutOrStdout(), cc.Items)
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(cc.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(cc.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline config history entries. 0 means no limit")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineConfigHistory(w io.Writer, cc []types.PipelineConfig) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tAGE")
	for _, c := range cc {
		fmt.Fprintf(tw, "%s\t%s\n", c.ID, formatters.FmtTime(c.CreatedAt))
	}
	tw.Flush()
}
