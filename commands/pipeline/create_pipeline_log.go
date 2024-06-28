package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreatePipelineLog(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var lines int
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:   "pipeline_log",
		Short: "Create a new log request within a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := config.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			params := cloud.CreatePipelineLog{
				PipelineID: pipelineID,
			}

			if lines > 0 {
				params.Lines = lines
			}

			out, err := config.Cloud.CreatePipelineLog(ctx, params)
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, out)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", out.ID, formatters.FmtTime(out.CreatedAt))
				tw.Flush()

				return nil
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Pipeline ID or name")
	fs.IntVar(&lines, "lines", 100, "Lines of logs to retrieve from the cluster")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("pipeline")
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.Completer.CompletePipelines)

	return cmd
}
