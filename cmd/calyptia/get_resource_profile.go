package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetResourceProfiles(config *config) *cobra.Command {
	var aggregatorKey string
	var last uint64
	var format string
	var showIDs bool
	var environment string

	cmd := &cobra.Command{
		Use:   "resource_profiles",
		Short: "Display latest resource profiles from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			aggregatorID, err := config.loadAggregatorID(aggregatorKey, environmentID)
			if err != nil {
				return err
			}

			pp, err := config.cloud.ResourceProfiles(config.ctx, aggregatorID, cloud.ResourceProfilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your resource profiles: %w", err)
			}

			switch format {
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
					fmt.Fprintf(tw, "%s\t%d\t%v\t%s\t%s\t%v\t%d\t%s\t%s\t%s\t%s\t%s\n", p.Name, p.StorageMaxChunksUp, p.StorageSyncFull, p.StorageBacklogMemLimit, p.StorageVolumeSize, p.StorageMaxChunksPause, p.CPUBufferWorkers, p.CPULimit, p.CPURequest, p.MemoryLimit, p.MemoryRequest, fmtAgo(p.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your resource profiles: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&aggregatorKey, "aggregator", "", "Parent aggregator ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include resource profile IDs in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("aggregator", config.completeAggregators)

	_ = cmd.MarkFlagRequired("aggregator") // TODO: use default aggregator ID from config cmd.

	return cmd
}

func (config *config) completeResourceProfiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// TODO: complete resource profiles.
	return []string{
		cloud.ResourceProfileHighPerformanceGuaranteedDelivery,
		cloud.ResourceProfileHighPerformanceOptimalThroughput,
		cloud.ResourceProfileBestEffortLowResource,
	}, cobra.ShellCompDirectiveNoFileComp
}
