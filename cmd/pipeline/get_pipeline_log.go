package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetPipelineLogs(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "pipeline_logs",
		Short: "Get pipeline logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := completer.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.Cloud.PipelineLogs(config.Ctx, cloud.ListPipelineLogs{
				PipelineID: pipelineID,
				Last:       &last,
			})

			if err != nil {
				return fmt.Errorf("could not fetch pipeline logs: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, ff.Items)
			}

			switch outputFormat {
			case "table":
				renderPipelineLogs(cmd.OutOrStdout(), ff.Items, showIDs)
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline files. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func NewCmdGetPipelineLog(c *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline_log PIPELINE_LOGS_ID",
		Short: "Get a specific pipeline log",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			check, err := c.Cloud.PipelineLog(c.Ctx, id)
			if err != nil {
				return err
			}
			fmt.Println(check.Logs)
			return nil
		},
	}
	return cmd
}

func renderPipelineLogs(w io.Writer, ff []cloud.PipelineLog, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "STATUS\tLINES\tAGE")
	for _, f := range ff {
		if showIDs {
			fmt.Fprintf(tw, "%s\t", f.ID)
		}
		fmt.Fprintf(tw, "%s\t%v\t%s\n", f.Status, f.Lines, formatters.FmtTime(f.CreatedAt))
	}
	tw.Flush()
}
