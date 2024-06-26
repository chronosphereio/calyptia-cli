package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
	"github.com/calyptia/cli/pointer"
)

func NewCmdUpdatePipeline(cfg *config.Config) *cobra.Command {
	var newName string
	var newConfigFile string
	var newReplicasCount int
	var noAutoCreateEndpointsFromConfig bool
	var skipConfigValidation bool
	var secretsFile string
	var secretsFormat string
	var files []string
	var encryptFiles bool
	var image string
	var metadataPairs []string
	var metadataFile string
	var providedConfigFormat string
	var deploymentStrategy string
	var portsServiceType string
	var minReplicas int32
	var scaleUpType string
	var scaleUpValue int32
	var scaleUpPeriodSeconds int32
	var scaleDownType string
	var scaleDownValue int32
	var scaleDownPeriodSeconds int32
	var utilizationCPUAverage int32
	var utilizationMemoryAverage int32

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Update a single pipeline by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompletePipelines,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			fs := cmd.Flags()
			if fs.Changed("scale-up-type") {
				if scaleUpType != "" {
					if scaleUpPeriodSeconds == 0 {
						return fmt.Errorf("invalid scale up policy - scale-up-period-seconds must be greater than zero")
					}
					if scaleUpValue == 0 {
						return fmt.Errorf("invalid scale up policy - scale-up-value must be greater than zero")
					}
				}
			}

			if fs.Changed("scale-down-type") {
				if scaleDownType != "" {
					if scaleDownPeriodSeconds == 0 {
						return fmt.Errorf("invalid scale down policy - scale-down-period-seconds must be greater than zero")
					}
					if scaleDownValue == 0 {
						return fmt.Errorf("invalid scale down policy - scale-down-value must be greater than zero")
					}
				}
			}

			if fs.Changed("utilization-cpu-average") {
				if utilizationCPUAverage <= 0 {
					return fmt.Errorf("utilization-cpu-average must be greater than zero")
				}
			}

			if fs.Changed("utilization-memory-average") {
				if utilizationMemoryAverage <= 0 {
					return fmt.Errorf("utilization-memory-average must be greater than zero")
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fs := cmd.Flags()

			var rawConfig string
			if newConfigFile != "" {
				b, err := os.ReadFile(newConfigFile)
				if err != nil {
					return fmt.Errorf("could not read config file: %w", err)
				}

				rawConfig = string(b)
			}

			secrets, err := parseUpdatePipelineSecrets(secretsFile, secretsFormat)
			if err != nil {
				return err
			}

			var updatePipelineFiles []cloudtypes.UpdatePipelineFile
			for _, f := range files {
				if f == "" {
					continue
				}

				name := filepath.Base(f)
				name = strings.TrimSuffix(name, filepath.Ext(name))
				// TODO: better sanitize file name.
				contents, err := os.ReadFile(f)
				if err != nil {
					return fmt.Errorf("coult not read file %q: %w", f, err)
				}

				updatePipelineFiles = append(updatePipelineFiles, cloudtypes.UpdatePipelineFile{
					Name:      &name,
					Contents:  &contents,
					Encrypted: &encryptFiles,
				})
			}

			var metadata *json.RawMessage
			if metadataFile != "" {
				b, err := os.ReadFile(metadataFile)
				if err != nil {
					return fmt.Errorf("could not read metadata file: %w", err)
				}

				metadata = &json.RawMessage{}
				*metadata = b
			} else {
				metadata, err = parseMetadataPairs(metadataPairs)
				if err != nil {
					return fmt.Errorf("could not parse metadata: %w", err)
				}
			}

			pipelineKey := args[0]
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			var format cloudtypes.ConfigFormat

			if providedConfigFormat != "" {
				format = cloudtypes.ConfigFormat(providedConfigFormat)
			} else if rawConfig != "" {
				// infer the configuration format from the file.
				format, err = InferConfigFormat(newConfigFile)
				if err != nil {
					return err
				}
			} else {
				format = cloudtypes.ConfigFormatINI
			}

			sut := cloudtypes.HPAScalingPolicyType(scaleUpType)
			sdt := cloudtypes.HPAScalingPolicyType(scaleDownType)

			update := cloudtypes.UpdatePipeline{
				NoAutoCreateEndpointsFromConfig: noAutoCreateEndpointsFromConfig,
				SkipConfigValidation:            skipConfigValidation,
				ConfigFormat:                    &format,
				Secrets:                         secrets,
				Files:                           updatePipelineFiles,
				Metadata:                        metadata,
			}

			if fs.Changed("min-replicas") {
				update.MinReplicas = &minReplicas
			}

			if fs.Changed("scale-up-type") {
				if scaleUpType != "" {
					update.ScaleUpType = &sut
					update.ScaleUpValue = &scaleUpValue
					update.ScaleUpPeriodSeconds = &scaleUpPeriodSeconds
				}
			}

			if fs.Changed("scale-down-type") {
				if scaleDownType != "" {
					update.ScaleDownType = &sdt
					update.ScaleDownValue = &scaleDownValue
					update.ScaleDownPeriodSeconds = &scaleDownPeriodSeconds
				}
			}

			if fs.Changed("utilization-cpu-average") {
				if utilizationCPUAverage > 0 {
					update.UtilizationCPUAverage = &utilizationCPUAverage
				}
			}

			if fs.Changed("utilization-memory-average") {
				if utilizationMemoryAverage > 0 {
					update.UtilizationMemoryAverage = &utilizationMemoryAverage
				}
			}

			ports, err := cfg.Cloud.PipelinePorts(ctx, pipelineID, cloudtypes.PipelinePortsParams{})
			if err != nil {
				return fmt.Errorf("could not update pipeline: %w", err)
			}

			var currentPortKind string
			if len(ports.Items) > 0 {
				currentPortKind = string(ports.Items[0].Kind)
			}

			if portsServiceType != "" {
				if !formatters.ValidPortKind(portsServiceType) {
					return fmt.Errorf("invalid provided service type %s, options are: %s", portsServiceType, formatters.PortKinds())
				}
				k := cloudtypes.PipelinePortKind(portsServiceType)
				update.PortKind = &k
			} else if currentPortKind != "" {
				k := cloudtypes.PipelinePortKind(currentPortKind)
				update.PortKind = &k
			}

			var strategy *cloudtypes.DeploymentStrategy
			if deploymentStrategy != "" {
				if !isValidDeploymentStrategy(deploymentStrategy) {
					return fmt.Errorf("invalid provided deployment strategy: %s", deploymentStrategy)
				}
				s := cloudtypes.DeploymentStrategy(deploymentStrategy)
				strategy = &s
			}

			if strategy != nil {
				update.DeploymentStrategy = strategy
			}

			if newName != "" {
				update.Name = &newName
			}
			if newReplicasCount >= 0 {
				update.ReplicasCount = pointer.From(uint(newReplicasCount))
			}

			if rawConfig != "" {
				update.RawConfig = &rawConfig
			}
			if image != "" {
				update.Image = &image
			}

			updated, err := cfg.Cloud.UpdatePipeline(ctx, pipelineID, update)
			if err != nil {
				return fmt.Errorf("could not update pipeline: %w", err)
			}

			if noAutoCreateEndpointsFromConfig && len(updated.AddedPorts) != 0 {
				fs := cmd.Flags()
				outputFormat := formatters.OutputFormatFromFlags(fs)
				if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
					return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), updated)
				}

				switch outputFormat {
				case "json":
					return json.NewEncoder(cmd.OutOrStdout()).Encode(updated)
				case "yml", "yaml":
					return yaml.NewEncoder(cmd.OutOrStdout()).Encode(updated)
				default:
					tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
					fmt.Fprintln(tw, "PROTOCOL\tFRONTEND-PORT\tBACKEND-PORT")
					for _, p := range updated.AddedPorts {
						fmt.Fprintf(tw, "%s\t%d\t%d\n", p.Protocol, p.FrontendPort, p.BackendPort)
					}
					return tw.Flush()
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&newName, "new-name", "", "New pipeline name")
	fs.StringVar(&newConfigFile, "config-file", "", "New Fluent Bit config file used by pipeline")
	fs.StringVar(&providedConfigFormat, "config-format", "", "Default configuration format to use (yaml, ini(deprecated))")
	fs.IntVar(&newReplicasCount, "replicas", -1, "New pipeline replica size")
	fs.BoolVar(&noAutoCreateEndpointsFromConfig, "disable-auto-ports", false, "Disables automatically creating ports from the config file if updated")
	fs.StringVar(&portsServiceType, "service-type", "", fmt.Sprintf("Service type to use for all ports that are auto-created on this pipeline, options are: %s", formatters.PortKinds()))
	fs.BoolVar(&skipConfigValidation, "skip-config-validation", false, "Opt-in to skip config validation (Use with caution as this option might be removed soon)")
	fs.StringVar(&secretsFile, "secrets-file", "", "Optional file containing a full definition of all secrets.\nThe format is derived either from the extension or the --secrets-format argument.\nThese can be referenced in pipeline files as such:\n{{ secrets.name }}\nThe prefix is the same for all secrets, the name is defined in the secrets file.")
	fs.StringVar(&secretsFormat, "secrets-format", "auto", "Secrets file format. Allowed: auto, env, json, yaml. If not set it is derived from secrets file extension")
	fs.StringVar(&deploymentStrategy, "deployment-strategy", "", "The deployment strategy to use when deploying this pipeline in cluster (hotReload or recreate (default)).")
	fs.StringArrayVar(&files, "file", nil, "Optional file. You can reference this file contents from your config like so:\n{{ files.myfile }}\nPass as many as you want; bear in mind the file name can only contain alphanumeric characters.")
	fs.BoolVar(&encryptFiles, "encrypt-files", false, "Encrypt file contents")
	fs.StringVar(&image, "image", "", "Fluent-bit docker image")
	fs.StringSliceVar(&metadataPairs, "metadata", nil, "Metadata to attach to the pipeline in the form of key:value. You could instead use a file with the --metadata-file option")
	fs.StringVar(&metadataFile, "metadata-file", "", "Metadata JSON file to attach to the pipeline intead of passing multiple --metadata flags")
	formatters.BindFormatFlags(cmd)

	// HPA parameters
	fs.Int32Var(&minReplicas, "min-replicas", 0, "Minimum replicas count for HPA")
	fs.StringVar(&scaleUpType, "scale-up-type", "", "The type of the policy which could be used while making scaling decisions. Accepted values Pods or Percent")
	fs.Int32Var(&scaleUpValue, "scale-up-value", 0, "Value contains the amount of change which is permitted by the scale up policy. Must be greater than 0")
	fs.Int32Var(&scaleUpPeriodSeconds, "scale-up-period-seconds", 0, "PeriodSeconds specifies the window of time for which the scale up policy should hold true.")
	fs.StringVar(&scaleDownType, "scale-down-type", "", "The type of the policy which could be used while making scaling decisions. Accepted values Pods or Percent")
	fs.Int32Var(&scaleDownValue, "scale-down-value", 0, "Value contains the amount of change which is permitted by the scale down policy. Must be greater than 0")
	fs.Int32Var(&scaleDownPeriodSeconds, "scale-down-period-seconds", 0, "PeriodSeconds specifies the window of time for which the scale down policy should hold true.")
	fs.Int32Var(&utilizationCPUAverage, "utilization-cpu-average", 0, "UtilizationCPUAverage defines the target percentage value for average CPU utilization.")
	fs.Int32Var(&utilizationMemoryAverage, "utilization-memory-average", 0, "UtilizationCPUAverage defines the target percentage value for average memory utilization.")

	return cmd
}

func parseUpdatePipelineSecrets(file, format string) ([]cloudtypes.UpdatePipelineSecret, error) {
	if file == "" {
		return nil, nil
	}

	b, err := readFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read secrets file: %w", err)
	}

	if format == "" || format == "auto" {
		switch filepath.Ext(file) {
		case ".env":
			format = "env"
		case ".json":
			format = "json"
		case ".yaml", ".yml":
			format = "yaml"
		default:
			return nil, fmt.Errorf("could not determine secrets format: %q", file)
		}
	}

	var secrets []cloudtypes.UpdatePipelineSecret
	switch format {
	case "env", "dotenv":
		m, err := godotenv.Parse(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloudtypes.UpdatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloudtypes.UpdatePipelineSecret{
				Key:   &k,
				Value: pointer.From([]byte(v)),
			})
		}
	case "json":
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloudtypes.UpdatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloudtypes.UpdatePipelineSecret{
				Key:   &k,
				Value: pointer.From([]byte(fmt.Sprintf("%v", v))),
			})
		}
	case "yml", "yaml":
		var m map[string]interface{}
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloudtypes.UpdatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloudtypes.UpdatePipelineSecret{
				Key:   &k,
				Value: pointer.From([]byte(fmt.Sprintf("%v", v))),
			})
		}
	}

	return secrets, nil
}
