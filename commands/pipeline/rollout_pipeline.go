package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdRolloutPipeline(cfg *config.Config) *cobra.Command {
	var stepsBack uint
	var toConfigID string
	var noAutoCreateEndpointsFromConfig, skipConfigValidation bool
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompletePipelines,
		Short:             "Rollout a pipeline to a previous config",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineKey := args[0]
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			var rawConfig string
			if toConfigID != "" {
				hh, err := cfg.Cloud.PipelineConfigHistory(ctx, pipelineID, cloudtypes.PipelineConfigHistoryParams{})
				if err != nil {
					return err
				}

				for _, c := range hh.Items {
					if c.ID == toConfigID {
						rawConfig = c.RawConfig
						break
					}
				}

				if rawConfig == "" {
					return fmt.Errorf("could not find config %q", toConfigID)
				}
			} else if stepsBack > 0 {
				hh, err := cfg.Cloud.PipelineConfigHistory(ctx, pipelineID, cloudtypes.PipelineConfigHistoryParams{
					Last: &stepsBack,
				})
				if err != nil {
					return err
				}

				if len(hh.Items) < int(stepsBack) {
					return fmt.Errorf("not enough history to rollback %d steps", stepsBack)
				}

				rawConfig = hh.Items[stepsBack-1].RawConfig
			} else {
				return fmt.Errorf("no config specified")
			}

			updated, err := cfg.Cloud.UpdatePipeline(ctx, pipelineID, cloudtypes.UpdatePipeline{
				RawConfig:                       &rawConfig,
				NoAutoCreateEndpointsFromConfig: noAutoCreateEndpointsFromConfig,
				SkipConfigValidation:            skipConfigValidation,
			})
			if err != nil {
				return err
			}

			if noAutoCreateEndpointsFromConfig && len(updated.AddedPorts) != 0 {
				if strings.HasPrefix(outputFormat, "go-template") {
					return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, updated)
				}

				switch outputFormat {
				case "table":
					tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
					fmt.Fprintln(tw, "PROTOCOL\tFRONTEND-PORT\tBACKEND-PORT")
					for _, p := range updated.AddedPorts {
						fmt.Fprintf(tw, "%s\t%d\t%d\n", p.Protocol, p.FrontendPort, p.BackendPort)
					}
					tw.Flush()
				case "json":
					return json.NewEncoder(cmd.OutOrStdout()).Encode(updated)
				case "yml", "yaml":
					return yaml.NewEncoder(cmd.OutOrStdout()).Encode(updated)
				default:
					return fmt.Errorf("unknown output format %q", outputFormat)
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.UintVar(&stepsBack, "steps-back", 1, "Steps back to rollout")
	fs.StringVar(&toConfigID, "to-config-id", "", "Configuration ID to rollout to. It overrides steps-back")
	fs.BoolVar(&noAutoCreateEndpointsFromConfig, "disable-auto-ports", false, "Disables automatically creating ports from the config file")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")
	fs.BoolVar(&skipConfigValidation, "skip-config-validation", false, "Opt-in to skip config validation (Use with caution as this option might be removed soon)")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
