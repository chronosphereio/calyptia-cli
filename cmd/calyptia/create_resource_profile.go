package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
)

type ResourceProfileSpec struct {
	Resources struct {
		Storage struct {
			SyncFull        bool   `json:"syncFull"`
			BacklogMemLimit string `json:"backlogMemLimit"`
			VolumeSize      string `json:"volumeSize"`
			MaxChunksUp     uint   `json:"maxChunksUp"`
			MaxChunksPause  bool   `json:"maxChunksPause"`
		} `json:"storage"`
		CPU struct {
			BufferWorkers uint   `json:"bufferWorkers"`
			Limit         string `json:"limit"`
			Request       string `json:"request"`
		} `json:"cpu"`
		Memory struct {
			Limit   string `json:"limit"`
			Request string `json:"request"`
		} `json:"memory"`
	} `json:"resources"`
}

var resourceProfileSpecExample = func() string {
	b, err := json.MarshalIndent(ResourceProfileSpec{}, "", "  ")
	if err != nil {
		panic("failed to marshal example spec")
	}

	return string(b)
}()

func newCmdCreateResourceProfile(config *config) *cobra.Command {
	var aggregatorKey string
	var name string
	var specFile string
	var outputFormat string
	cmd := &cobra.Command{
		Use:   "resource_profile",
		Short: "Create a new resource profile attached to an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			rawSpec, err := readFile(specFile)
			if err != nil {
				return fmt.Errorf("could not read config file: %w", err)
			}

			var spec ResourceProfileSpec
			err = json.Unmarshal(rawSpec, &spec)
			if err != nil {
				return fmt.Errorf("could not parse json spec: %w", err)
			}

			aggregatorID, err := config.loadAggregatorID(aggregatorKey)
			if err != nil {
				return err
			}

			rp, err := config.cloud.CreateResourceProfile(config.ctx, aggregatorID, cloud.CreateResourceProfile{
				Name:                   name,
				StorageMaxChunksUp:     spec.Resources.Storage.MaxChunksUp,
				StorageSyncFull:        spec.Resources.Storage.SyncFull,
				StorageBacklogMemLimit: spec.Resources.Storage.BacklogMemLimit,
				StorageVolumeSize:      spec.Resources.Storage.VolumeSize,
				StorageMaxChunksPause:  spec.Resources.Storage.MaxChunksPause,
				CPUBufferWorkers:       spec.Resources.CPU.BufferWorkers,
				CPULimit:               spec.Resources.CPU.Limit,
				CPURequest:             spec.Resources.CPU.Request,
				MemoryLimit:            spec.Resources.Memory.Limit,
				MemoryRequest:          spec.Resources.Memory.Request,
			})
			if err != nil {
				return fmt.Errorf("could not create resource profile: %w", err)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", rp.ID, fmtAgo(rp.CreatedAt))
				tw.Flush()
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(rp)
				if err != nil {
					return fmt.Errorf("could not json encode your new resource profile: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&aggregatorKey, "aggregator", "", "Parent aggregator ID or name")
	fs.StringVar(&name, "name", "", "Resource profile name")
	fs.StringVar(&specFile, "spec", "", "Take spec from JSON file. Example:\n"+resourceProfileSpecExample)
	fs.StringVar(&outputFormat, "output-format", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("aggregator", config.completeAggregators)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.MarkFlagRequired("aggregator") // TODO: use default aggregator key from config cmd.
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("spec")

	return cmd
}
