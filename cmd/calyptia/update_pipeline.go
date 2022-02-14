package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
)

func newCmdUpdatePipeline(config *config) *cobra.Command {
	var newName string
	var newConfigFile string
	var newReplicasCount uint64
	var autoCreatePortsFromConfig bool
	var secretsFile string
	var secretsFormat string
	var files []string
	var encryptFiles bool
	var outputFormat string
	var metadataPairs []string
	var metadataFile string

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

			secrets, err := parseUpdatePipelineSecrets(secretsFile, secretsFormat)
			if err != nil {
				return err
			}

			var updatePipelineFiles []cloud.UpdatePipelineFile
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

				fmt.Println("encrypting file", encryptFiles)

				updatePipelineFiles = append(updatePipelineFiles, cloud.UpdatePipelineFile{
					Name:      &name,
					Contents:  &contents,
					Encrypted: &encryptFiles,
				})
			}

			var metadata *json.RawMessage
			if metadataFile != "" {
				b, err := readFile(metadataFile)
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
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			update := cloud.UpdatePipeline{
				AutoCreatePortsFromConfig: &autoCreatePortsFromConfig,
				Secrets:                   secrets,
				Files:                     updatePipelineFiles,
				Metadata:                  metadata,
			}
			if newName != "" {
				update.Name = &newName
			}
			if newReplicasCount != 0 {
				update.ReplicasCount = &newReplicasCount
			}
			if rawConfig != "" {
				update.RawConfig = &rawConfig
			}

			updated, err := config.cloud.UpdatePipeline(config.ctx, pipelineID, update)
			if err != nil {
				return fmt.Errorf("could not update pipeline: %w", err)
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
	fs.StringVar(&newName, "new-name", "", "New pipeline name")
	fs.StringVar(&newConfigFile, "config-file", "", "New Fluent Bit config file used by pipeline")
	fs.Uint64Var(&newReplicasCount, "replicas", 0, "New pipeline replica size")
	fs.BoolVar(&autoCreatePortsFromConfig, "auto-create-ports", true, "Automatically create pipeline ports from config if updated")
	fs.StringVar(&secretsFile, "secrets-file", "", "Optional file where secrets are defined. You can store key values and reference them inside your config like so:\n{{ secrets.foo }}")
	fs.StringVar(&secretsFormat, "secrets-format", "auto", "Secrets file format. Allowed: auto, env, json, yaml. Auto tries to detect it from file extension")
	fs.StringArrayVar(&files, "file", nil, "Optional file. You can reference this file contents from your config like so:\n{{ files.myfile }}\nPass as many as you want; bear in mind the file name can only contain alphanumeric characters.")
	fs.BoolVar(&encryptFiles, "encrypt-files", false, "Encrypt file contents")
	fs.StringSliceVar(&metadataPairs, "metadata", nil, "Metadata to attach to the pipeline in the form of key:value. You could instead use a file with the --metadata-file option")
	fs.StringVar(&metadataFile, "metadata-file", "", "Metadata JSON file to attach to the pipeline intead of passing multiple --metadata flags")
	fs.StringVar(&outputFormat, "output-format", "table", "Output format. Allowed: table, json")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func parseUpdatePipelineSecrets(file, format string) ([]cloud.UpdatePipelineSecret, error) {
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
			return nil, errors.Errorf("could not determine secrets format: %q", file)
		}
	}

	var secrets []cloud.UpdatePipelineSecret
	switch format {
	case "env", "dotenv":
		m, err := godotenv.Parse(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloud.UpdatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloud.UpdatePipelineSecret{
				Key:   &k,
				Value: ptrBytes([]byte(v)),
			})
		}
	case "json":
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloud.UpdatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloud.UpdatePipelineSecret{
				Key:   &k,
				Value: ptrBytes([]byte(fmt.Sprintf("%v", v))),
			})
		}
	case "yaml", "yml":
		var m map[string]interface{}
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloud.UpdatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloud.UpdatePipelineSecret{
				Key:   &k,
				Value: ptrBytes([]byte(fmt.Sprintf("%v", v))),
			})
		}
	}

	return secrets, nil
}

func ptrBytes(v []byte) *[]byte {
	return &v
}
