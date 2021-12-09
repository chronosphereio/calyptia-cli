package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdRolloutPipeline(config *config) *cobra.Command {
	var stepsBack uint64
	var toConfigID string
	var autoCreatePortsFromConfig bool
	var outputFormat string

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
				h, err := config.cloud.PipelineConfigHistory(config.ctx, pipelineID, 0)
				if err != nil {
					return err
				}

				for _, c := range h {
					if c.ID == toConfigID {
						rawConfig = c.RawConfig
						break
					}
				}

				if rawConfig == "" {
					return fmt.Errorf("could not find config %q", toConfigID)
				}
			} else if stepsBack > 0 {
				h, err := config.cloud.PipelineConfigHistory(config.ctx, pipelineID, stepsBack)
				if err != nil {
					return err
				}

				if len(h) < int(stepsBack) {
					return fmt.Errorf("not enough history to rollback %d steps", stepsBack)
				}

				rawConfig = h[stepsBack-1].RawConfig
			} else {
				return fmt.Errorf("no config specified")
			}

			fmt.Println(rawConfig)

			updated, err := config.cloud.UpdateAggregatorPipeline(config.ctx, pipelineID, cloud.UpdateAggregatorPipelineOpts{
				RawConfig:                 &rawConfig,
				AutoCreatePortsFromConfig: autoCreatePortsFromConfig,
			})
			if err != nil {
				return err
			}

			if autoCreatePortsFromConfig && len(updated.AddedPorts) != 0 {
				switch outputFormat {
				case "table":
					tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
					fmt.Fprintln(tw, "PROTOCOL\tFRONTEND-PORT\tBACKEND-PORT")
					for _, p := range updated.AddedPorts {
						fmt.Fprintf(tw, "%s\t%d\t%d\n", p.Protocol, p.FrontendPort, p.BackendPort)
					}
					tw.Flush()
				case "json":
					err := json.NewEncoder(os.Stdout).Encode(updated)
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
	fs.Uint64Var(&stepsBack, "steps-back", 1, "Steps back to rollout")
	fs.StringVar(&toConfigID, "to-config-id", "", "Configuration ID to rollout to. It overrides steps-back")
	fs.BoolVar(&autoCreatePortsFromConfig, "auto-create-ports", true, "Automatically create pipeline ports from config")
	fs.StringVar(&outputFormat, "output-format", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
