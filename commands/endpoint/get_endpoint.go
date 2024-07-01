package endpoint

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetEndpoints(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "endpoints",
		Short: "Display latest endpoints from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			pp, err := cfg.Cloud.PipelinePorts(ctx, pipelineID, cloudtypes.PipelinePortsParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline endpoints: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), pp)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			default:
				return formatters.RenderEndpointsTable(cmd.OutOrStdout(), pp.Items, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline endpoints. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include endpoint IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
