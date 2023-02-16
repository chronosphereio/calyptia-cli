package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdGetResourceProfiles(config *cfg.Config) *cobra.Command {
	var coreInstanceKey string
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool
	var environment string

	cmd := &cobra.Command{
		Use:   "resource_profiles",
		Short: "Display latest resource profiles from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			coreInstanceID, err := config.LoadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			pp, err := config.Cloud.ResourceProfiles(config.Ctx, coreInstanceID, cloud.ResourceProfilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your resource profiles: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, pp.Items)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tSTORAGE-MAX-CHUNKS-UP\tSTORAGE-SYNC-FULL\tSTORAGE-BACKLOG-MEM-LIMIT\tSTORAGE-VOLUME-SIZE\tSTORAGE-MAX-CHUNKS-PAUSE\tCPU-BUFFER-WORKERS\tCPU-LIMIT\tCPU-REQUEST\tMEM-LIMIT\tMEM-REQUEST\tAGE")
				for _, p := range pp.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", p.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%v\t%s\t%s\t%v\t%d\t%s\t%s\t%s\t%s\t%s\n", p.Name, p.StorageMaxChunksUp, p.StorageSyncFull, p.StorageBacklogMemLimit, p.StorageVolumeSize, p.StorageMaxChunksPause, p.CPUBufferWorkers, p.CPULimit, p.CPURequest, p.MemoryLimit, p.MemoryRequest, fmtTime(p.CreatedAt))
				}
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include resource profile IDs in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", config.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default aggregator ID from config cmd.

	return cmd
}
