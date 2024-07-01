package pipeline

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetPipelineStatusHistory(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_status_history",
		Short: "Display latest status history from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			ss, err := cfg.Cloud.PipelineStatusHistory(ctx, pipelineID, cloudtypes.PipelineStatusHistoryParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline status history: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ss.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "STATUS\tCONFIG-ID\tAGE")
				for _, s := range ss.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", s.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", s.Status, s.Config.ID, formatters.FmtTime(s.CreatedAt))
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline status history entries. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
