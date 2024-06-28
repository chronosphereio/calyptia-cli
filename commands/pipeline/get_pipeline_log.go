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

func NewCmdGetPipelineLogs(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_logs",
		Short: "Get pipeline logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.PipelineLogs(ctx, cloudtypes.ListPipelineLogs{
				PipelineID: pipelineID,
				Last:       &last,
			})

			if err != nil {
				return fmt.Errorf("could not fetch pipeline logs: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ff.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			default:
				return renderPipelineLogs(cmd.OutOrStdout(), ff.Items, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline files. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func NewCmdGetPipelineLog(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline_log PIPELINE_LOGS_ID",
		Short: "Get a specific pipeline log",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]
			check, err := cfg.Cloud.PipelineLog(ctx, id)
			if err != nil {
				return err
			}
			fmt.Println(check.Logs)
			return nil
		},
	}
	return cmd
}

func renderPipelineLogs(w io.Writer, ff []cloudtypes.PipelineLog, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(tw, "STATUS\tLINES\tAGE"); err != nil {
		return err
	}

	for _, f := range ff {
		if showIDs {
			if _, err := fmt.Fprintf(tw, "%s\t", f.ID); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%v\t%s\n", f.Status, f.Lines, formatters.FmtTime(f.CreatedAt))
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}
