package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdRolloutPipeline(config *config) *cobra.Command {
	var stepsBack uint
	var toConfigID string
	var autoCreatePortsFromConfig bool
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		Short:             "Rollout a pipeline to a previous config",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			var rawConfig string
			if toConfigID != "" {
				hh, err := config.cloud.PipelineConfigHistory(config.ctx, pipelineID, cloud.PipelineConfigHistoryParams{})
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
				hh, err := config.cloud.PipelineConfigHistory(config.ctx, pipelineID, cloud.PipelineConfigHistoryParams{
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

			updated, err := config.cloud.UpdatePipeline(config.ctx, pipelineID, cloud.UpdatePipeline{
				RawConfig:                 &rawConfig,
				AutoCreatePortsFromConfig: &autoCreatePortsFromConfig,
			})
			if err != nil {
				return err
			}

			if autoCreatePortsFromConfig && len(updated.AddedPorts) != 0 {
				if strings.HasPrefix(outputFormat, "go-template") {
					return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, updated)
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
					err := json.NewEncoder(cmd.OutOrStdout()).Encode(updated)
					if err != nil {
						return fmt.Errorf("could not json encode updated pipeline: %w", err)
					}
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

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
