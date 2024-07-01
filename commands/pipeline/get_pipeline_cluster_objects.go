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

func NewCmdGetPipelineClusterObjects(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_objects",
		Short: "Get pipeline cluster objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			co, err := cfg.Cloud.PipelineClusterObjects(ctx, pipelineID, cloudtypes.PipelineClusterObjectsParams{
				Last: &last,
			})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), co.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(co.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(co.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tKIND\tCREATED AT")
				for _, c := range co.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", c.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", c.Name, string(c.Kind), formatters.FmtTime(c.CreatedAt))
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Pipeline to list cluster objects for")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` cluster objects. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}
