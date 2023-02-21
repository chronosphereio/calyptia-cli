package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdRolloutPipeline(config *cfg.Config) *cobra.Command {
	var stepsBack uint
	var toConfigID string
	var autoCreatePortsFromConfig, skipConfigValidation bool
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompletePipelines,
		Short:             "Rollout a pipeline to a previous config",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			pipelineID, err := completer.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			var rawConfig string
			if toConfigID != "" {
				hh, err := config.Cloud.PipelineConfigHistory(config.Ctx, pipelineID, cloud.PipelineConfigHistoryParams{})
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
				hh, err := config.Cloud.PipelineConfigHistory(config.Ctx, pipelineID, cloud.PipelineConfigHistoryParams{
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

			updated, err := config.Cloud.UpdatePipeline(config.Ctx, pipelineID, cloud.UpdatePipeline{
				RawConfig:                 &rawConfig,
				AutoCreatePortsFromConfig: &autoCreatePortsFromConfig,
				SkipConfigValidation:      skipConfigValidation,
			})
			if err != nil {
				return err
			}

			if autoCreatePortsFromConfig && len(updated.AddedPorts) != 0 {
				if strings.HasPrefix(outputFormat, "go-template") {
					return utils.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, updated)
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
	fs.BoolVar(&autoCreatePortsFromConfig, "auto-create-ports", true, "Automatically create pipeline ports from config")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")
	fs.BoolVar(&skipConfigValidation, "skip-config-validation", false, "Opt-in to skip config validation (Use with caution as this option might be removed soon)")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
