package tracesession

import (
	"encoding/json"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateTraceSession(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var plugins []string
	var lifespan time.Duration
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:   "trace_session", // child of `create`
		Short: "Create trace session",
		Long: "Start a new trace session on the given pipeline.\n" +
			"There can only be one active trace session at a moment.\n" +
			"Either terminate the current active one and create a new one,\n" +
			"or update it and extend its lifespan.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			created, err := cfg.Cloud.CreateTraceSession(ctx, pipelineID, cloudtypes.CreateTraceSession{
				Plugins:  plugins,
				Lifespan: cloudtypes.Duration(lifespan),
			})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), created)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				return formatters.RenderCreated(cmd.OutOrStdout(), created)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline (name or ID) in which to start the trace session")
	fs.StringSliceVar(&plugins, "plugins", nil, "Fluent-bit plugins to trace")
	fs.DurationVar(&lifespan, "lifespan", time.Minute*10, "Trace session lifespan")
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("pipeline")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("plugins", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := cmd.Context()
		return cfg.Completer.CompletePipelinePlugins(ctx, pipelineKey, cmd, args, toComplete)
	})

	return cmd
}
