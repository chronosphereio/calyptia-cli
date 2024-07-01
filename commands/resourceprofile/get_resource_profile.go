package resourceprofile

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

func NewCmdGetResourceProfiles(cfg *config.Config) *cobra.Command {
	var coreInstanceKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "resource_profiles",
		Short: "Display latest resource profiles from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey)
			if err != nil {
				return err
			}

			pp, err := cfg.Cloud.ResourceProfiles(ctx, coreInstanceID, cloudtypes.ResourceProfilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your resource profiles: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), pp.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tSTORAGE-MAX-CHUNKS-UP\tSTORAGE-SYNC-FULL\tSTORAGE-BACKLOG-MEM-LIMIT\tSTORAGE-VOLUME-SIZE\tSTORAGE-MAX-CHUNKS-PAUSE\tCPU-BUFFER-WORKERS\tCPU-LIMIT\tCPU-REQUEST\tMEM-LIMIT\tMEM-REQUEST\tAGE")
				for _, p := range pp.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", p.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%v\t%s\t%s\t%v\t%d\t%s\t%s\t%s\t%s\t%s\n", p.Name, p.StorageMaxChunksUp, p.StorageSyncFull, p.StorageBacklogMemLimit, p.StorageVolumeSize, p.StorageMaxChunksPause, p.CPUBufferWorkers, p.CPULimit, p.CPURequest, p.MemoryLimit, p.MemoryRequest, formatters.FmtTime(p.CreatedAt))
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include resource profile IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default aggregator ID from config cmd.

	return cmd
}
