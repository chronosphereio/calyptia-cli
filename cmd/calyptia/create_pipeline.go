package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newCmdCreatePipeline(config *config) *cobra.Command {
	var aggregatorKey string
	var name string
	var image string
	var replicasCount uint64
	var configFile string
	var secretsFile string
	var secretsFormat string
	var files []string
	var encryptFiles bool
	var autoCreatePortsFromConfig bool
	var resourceProfileName string
	var outputFormat string
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Create a new pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: support `@INCLUDE`. See https://docs.fluentbit.io/manual/administration/configuring-fluent-bit/configuration-file#config_include_file-1
			rawConfig, err := readFile(configFile)
			if err != nil {
				return fmt.Errorf("could not read config file: %w", err)
			}

			var secrets []cloud.AddPipelineSecretPayload
			if secretsFile != "" {
				b, err := readFile(secretsFile)
				if err != nil {
					return fmt.Errorf("could not read secrets file: %w", err)
				}

				if secretsFormat == "" || secretsFormat == "auto" {
					switch filepath.Ext(secretsFile) {
					case ".env":
						secretsFormat = "env"
					case ".json":
						secretsFormat = "json"
					case ".yaml", ".yml":
						secretsFormat = "yaml"
					default:
						return errors.Errorf("could not determine secrets format: %q", secretsFile)
					}

					switch secretsFormat {
					case "env", "dotenv":
						m, err := godotenv.Parse(bytes.NewReader(b))
						if err != nil {
							return fmt.Errorf("could not parse secrets file %q: %w", secretsFile, err)
						}

						secrets = make([]cloud.AddPipelineSecretPayload, 0, len(m))
						for k, v := range m {
							secrets = append(secrets, cloud.AddPipelineSecretPayload{
								Key:   k,
								Value: []byte(v),
							})
						}
					case "json":
						var m map[string]interface{}
						if err := json.Unmarshal(b, &m); err != nil {
							return fmt.Errorf("could not parse secrets file %q: %w", secretsFile, err)
						}

						secrets = make([]cloud.AddPipelineSecretPayload, 0, len(m))
						for k, v := range m {
							secrets = append(secrets, cloud.AddPipelineSecretPayload{
								Key:   k,
								Value: []byte(fmt.Sprintf("%v", v)),
							})
						}
					case "yaml", "yml":
						var m map[string]interface{}
						if err := yaml.Unmarshal(b, &m); err != nil {
							return fmt.Errorf("could not parse secrets file %q: %w", secretsFile, err)
						}

						secrets = make([]cloud.AddPipelineSecretPayload, 0, len(m))
						for k, v := range m {
							secrets = append(secrets, cloud.AddPipelineSecretPayload{
								Key:   k,
								Value: []byte(fmt.Sprintf("%v", v)),
							})
						}
					}
				}
			}

			var addPipelineFilePayload []cloud.AddPipelineFilePayload
			for _, f := range files {
				if f == "" {
					continue
				}

				name := filepath.Base(f)
				name = strings.TrimSuffix(name, filepath.Ext(name))
				// TODO: better sanitize file name.
				contents, err := readFile(f)
				if err != nil {
					return fmt.Errorf("coult not read file %q: %w", f, err)
				}

				addPipelineFilePayload = append(addPipelineFilePayload, cloud.AddPipelineFilePayload{
					Name:      name,
					Contents:  contents,
					Encrypted: encryptFiles,
				})
			}

			aggregatorID, err := config.loadAggregatorID(aggregatorKey)
			if err != nil {
				return err
			}

			imagePtr := &image
			if image == "" {
				imagePtr = nil
			}
			a, err := config.cloud.CreateAggregatorPipeline(config.ctx, aggregatorID, cloud.AddAggregatorPipelinePayload{
				Name:                      name,
				ReplicasCount:             replicasCount,
				RawConfig:                 string(rawConfig),
				Image:                     imagePtr,
				Secrets:                   secrets,
				AutoCreatePortsFromConfig: autoCreatePortsFromConfig,
				ResourceProfile:           resourceProfileName,
				Files:                     addPipelineFilePayload,
			})
			if err != nil {
				if e, ok := err.(*cloud.Error); ok && e.Detail != nil {
					return errors.Errorf("could not create pipeline: %s: %s", err, *e.Detail)
				}

				return fmt.Errorf("could not create pipeline: %w", err)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "NAME\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", a.Name, fmtAgo(a.CreatedAt))
				tw.Flush()
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(a)
				if err != nil {
					return fmt.Errorf("could not json encode your new pipeline: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&aggregatorKey, "aggregator", "", "Parent aggregator ID or name")
	fs.StringVar(&name, "name", "", "Pipeline name; leave it empty to generate a random name")
	fs.StringVar(&image, "image", "", "Fluent-bit image to deploy")
	fs.Uint64Var(&replicasCount, "replicas", 1, "Pipeline replica size")
	fs.StringVar(&configFile, "config-file", "fluent-bit.conf", "Fluent Bit config file used by pipeline")
	fs.StringVar(&secretsFile, "secrets-file", "", "Optional file where secrets are defined. You can store key values and reference them inside your config like so:\n{{ secrets.foo }}")
	fs.StringVar(&secretsFormat, "secrets-format", "auto", "Secrets file format. Allowed: auto, env, json, yaml. Auto tries to detect it from file extension")
	fs.StringArrayVar(&files, "file", nil, "Optional file. You can reference this file contents from your config like so:\n{{ files.myfile }}\nPass as many as you want; bear in mind the file name can only contain alphanumeric characters.")
	fs.BoolVar(&encryptFiles, "encrypt-files", false, "Encrypt file contents")
	fs.BoolVar(&autoCreatePortsFromConfig, "auto-create-ports", true, "Automatically create pipeline ports from config")
	fs.StringVar(&resourceProfileName, "resource-profile", string(cloud.DefaultPipelineResourceProfile), "Resource profile name")
	fs.StringVar(&outputFormat, "output-format", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("aggregator", config.completeAggregators)
	_ = cmd.RegisterFlagCompletionFunc("secrets-format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "env", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("resource-profile", config.completeResourceProfiles)

	_ = cmd.MarkFlagRequired("aggregator") // TODO: use default aggregator key from config cmd.

	return cmd
}

func readFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read contents: %w", err)
	}

	return b, nil
}
