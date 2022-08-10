package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/calyptia/api/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newCmdCreateTraceSession(config *config) *cobra.Command {
	var pipelineKey string
	var plugins []string
	var lifespan time.Duration
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "trace_session", // child of `create`
		Short: "Create trace session",
		Long: "Start a new trace session on the given pipeline.\n" +
			"There can only be one active trace session at a moment.\n" +
			"Either terminate the current active one and create a new one,\n" +
			"or update it and extend its lifespan.",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			created, err := config.cloud.CreateTraceSession(config.ctx, pipelineID, types.CreateTraceSession{
				Plugins:  plugins,
				Lifespan: types.Duration(lifespan),
			})
			if err != nil {
				return err
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", created.ID, fmtAgo(created.CreatedAt))
				tw.Flush()

				return nil
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline (name or ID) in which to start the trace session")
	fs.StringSliceVar(&plugins, "plugins", nil, "Fluent-bit plugins to trace")
	fs.DurationVar(&lifespan, "lifespan", time.Minute*10, "Trace session lifespan")
	fs.StringVar(&outputFormat, "output", "table", "Output format (table, json, yaml)")

	_ = cmd.MarkFlagRequired("pipeline")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("plugins", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		pipelineID, err := config.loadPipelineID(pipelineKey)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		pipeline, err := config.cloud.Pipeline(config.ctx, pipelineID, types.PipelineParams{})
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		cfg, err := parsePipelineConfig(pipeline.Config)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		plugins := make([]string, 0, len(cfg.PluginIndex))
		for plugin := range cfg.PluginIndex {
			plugins = append(plugins, plugin)
		}

		return plugins, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
