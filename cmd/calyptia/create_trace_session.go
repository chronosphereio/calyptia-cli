package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/completer"
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdCreateTraceSession(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var plugins []string
	var lifespan time.Duration
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "trace_session", // child of `create`
		Short: "Create trace session",
		Long: "Start a new trace session on the given pipeline.\n" +
			"There can only be one active trace session at a moment.\n" +
			"Either terminate the current active one and create a new one,\n" +
			"or update it and extend its lifespan.",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			created, err := config.Cloud.CreateTraceSession(config.Ctx, pipelineID, types.CreateTraceSession{
				Plugins:  plugins,
				Lifespan: types.Duration(lifespan),
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, created)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", created.ID, fmtTime(created.CreatedAt))
				tw.Flush()

				return nil
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline (name or ID) in which to start the trace session")
	fs.StringSliceVar(&plugins, "plugins", nil, "Fluent-bit plugins to trace")
	fs.DurationVar(&lifespan, "lifespan", time.Minute*10, "Trace session lifespan")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("pipeline")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", completer.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("plugins", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return config.CompletePipelinePlugins(pipelineKey, cmd, args, toComplete)
	})

	return cmd
}
