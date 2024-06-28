package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetPipelineConfigHistory(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint

	cmd := &cobra.Command{
		Use:   "pipeline_config_history",
		Short: "Display latest config history from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			cc, err := cfg.Cloud.PipelineConfigHistory(ctx, pipelineID, cloudtypes.PipelineConfigHistoryParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline config history: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), cc)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(cc.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(cc.Items)
			default:
				return renderPipelineConfigHistory(cmd.OutOrStdout(), cc.Items)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline config history entries. 0 means no limit")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineConfigHistory(w io.Writer, cc []cloudtypes.PipelineConfig) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tAGE"); err != nil {
		return err
	}

	for _, c := range cc {
		_, err := fmt.Fprintf(tw, "%s\t%s\n", c.ID, formatters.FmtTime(c.CreatedAt))
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}
