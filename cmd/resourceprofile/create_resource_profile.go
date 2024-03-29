package resourceprofile

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
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

func NewCmdCreateResourceProfile(config *cfg.Config) *cobra.Command {
	var coreInstanceKey string
	var name string
	var specFile string
	var outputFormat, goTemplate string
	var environment string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "resource_profile",
		Short: "Create a new resource profile attached to a core-instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			rawSpec, err := cfg.ReadFile(specFile)
			if err != nil {
				return fmt.Errorf("could not read spec file: %w", err)
			}

			var spec ResourceProfileSpec
			err = json.Unmarshal(rawSpec, &spec)
			if err != nil {
				return fmt.Errorf("could not parse json spec: %w", err)
			}

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			aggregatorID, err := completer.LoadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			rp, err := config.Cloud.CreateResourceProfile(config.Ctx, aggregatorID, cloud.CreateResourceProfile{
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

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, rp)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", rp.ID, formatters.FmtTime(rp.CreatedAt))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(rp)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(rp)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.StringVar(&name, "name", "", "Resource profile name")
	fs.StringVar(&specFile, "spec", "", "Take spec from JSON file. Example:\n"+resourceProfileSpecExample)
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", completer.CompleteCoreInstances)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default core-instance key from config cmd.
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("spec")

	return cmd
}
