package resourceprofile

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateResourceProfile(cfg *config.Config) *cobra.Command {
	var coreInstanceKey string
	var name string
	var specFile string

	cmd := &cobra.Command{
		Use:   "resource_profile",
		Short: "Create a new resource profile attached to a core-instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			rawSpec, err := os.ReadFile(specFile)
			if err != nil {
				return fmt.Errorf("could not read spec file: %w", err)
			}

			var spec ResourceProfileSpec
			err = json.Unmarshal(rawSpec, &spec)
			if err != nil {
				return fmt.Errorf("could not parse json spec: %w", err)
			}

			aggregatorID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey)
			if err != nil {
				return err
			}

			rp, err := cfg.Cloud.CreateResourceProfile(ctx, aggregatorID, cloudtypes.CreateResourceProfile{
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

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), rp)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(rp)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(rp)
			default:
				return formatters.RenderCreated(cmd.OutOrStdout(), rp)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.StringVar(&name, "name", "", "Resource profile name")
	fs.StringVar(&specFile, "spec", "", "Take spec from JSON file. Example:\n"+resourceProfileSpecExample)
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)
	_ = cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default core-instance key from config cmd.
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("spec")

	return cmd
}
