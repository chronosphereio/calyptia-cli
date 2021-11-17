package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/calyptia/cloud"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdUpdatePipeline(config *config) *cobra.Command {
	var newName string
	var newConfigFile string
	var newReplicasCount uint64
	var autoCreatePortsFromConfig bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Update a single pipeline by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			var rawConfig string
			if newConfigFile != "" {
				b, err := readFile(newConfigFile)
				if err != nil {
					return fmt.Errorf("could not read config file: %w", err)
				}

				rawConfig = string(b)
			}

			pipelineKey := args[0]
			pipelineID := pipelineKey
			if !validUUID(pipelineID) {
				if pipelineKey == newName {
					return nil
				}

				aa, err := config.fetchAllPipelines()
				if err != nil {
					return err
				}

				a, ok := findPipelineByName(aa, pipelineKey)
				if !ok {
					return fmt.Errorf("could not find pipeline %q", pipelineKey)
				}

				pipelineID = a.ID
			}

			opts := cloud.UpdateAggregatorPipelineOpts{
				AutoCreatePortsFromConfig: autoCreatePortsFromConfig,
			}
			if newName != "" {
				opts.Name = &newName
			}
			if newReplicasCount != 0 {
				opts.ReplicasCount = &newReplicasCount
			}
			if rawConfig != "" {
				opts.RawConfig = &rawConfig
			}
			updated, err := config.cloud.UpdateAggregatorPipeline(config.ctx, pipelineID, opts)
			if err != nil {
				return fmt.Errorf("could not update pipeline: %w", err)
			}

			if autoCreatePortsFromConfig && len(updated.AddedPorts) != 0 {
				switch outputFormat {
				case "table":
					tw := table.NewWriter()
					tw.AppendHeader(table.Row{"Protocol", "Frontend port", "Backend port"})
					tw.Style().Options = table.OptionsNoBordersAndSeparators
					if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
						tw.SetAllowedRowLength(w)
					}
					for _, p := range updated.AddedPorts {
						tw.AppendRow(table.Row{p.Protocol, p.FrontendPort, p.BackendPort})
					}
					fmt.Println(tw.Render())
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
	fs.StringVar(&newName, "new-name", "", "New pipeline name")
	fs.StringVar(&newConfigFile, "config-file", "", "New Fluent Bit config file used by pipeline")
	fs.Uint64Var(&newReplicasCount, "replicas", 0, "New pipeline replica size")
	fs.BoolVar(&autoCreatePortsFromConfig, "auto-create-ports", true, "Automatically create pipeline ports from config if updated")
	fs.StringVar(&outputFormat, "output-format", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
