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

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdCreatePipeline(config *cfg.Config) *cobra.Command {
	var coreInstanceKey string
	var name string
	var replicasCount uint
	var configFile string
	var secretsFile string
	var secretsFormat string
	var files []string
	var encryptFiles bool
	var image string
	var autoCreatePortsFromConfig bool
	var skipConfigValidation bool
	var resourceProfileName string
	var outputFormat, goTemplate string
	var metadataPairs []string
	var metadataFile string
	var environment string

	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Create a new pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: support `@INCLUDE`. See https://docs.fluentbit.io/manual/administration/configuring-fluent-bit/configuration-file#config_include_file-1
			rawConfig, err := readFile(configFile)
			if err != nil {
				return fmt.Errorf("could not read config file: %w", err)
			}

			secrets, err := parseCreatePipelineSecret(secretsFile, secretsFormat)
			if err != nil {
				return fmt.Errorf("could not read secrets file: %w", err)
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

			var addFilesPayload []cloud.CreatePipelineFile
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

				addFilesPayload = append(addFilesPayload, cloud.CreatePipelineFile{
					Name:      name,
					Contents:  contents,
					Encrypted: encryptFiles,
				})
			}

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

			in := cloud.CreatePipeline{
				Name:                      name,
				ReplicasCount:             replicasCount,
				RawConfig:                 string(rawConfig),
				Secrets:                   secrets,
				AutoCreatePortsFromConfig: autoCreatePortsFromConfig,
				SkipConfigValidation:      skipConfigValidation,
				ResourceProfileName:       resourceProfileName,
				Files:                     addFilesPayload,
				Metadata:                  metadata,
			}

			if image != "" {
				in.Image = &image
			}

			a, err := config.Cloud.CreatePipeline(config.Ctx, coreInstanceID, in)
			if err != nil {
				if e, ok := err.(*cloud.Error); ok && e.Detail != nil {
					return fmt.Errorf("could not create pipeline: %s: %s", err, *e.Detail)
				}

				return fmt.Errorf("could not create pipeline: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, a)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tNAME\tAGE")
				fmt.Fprintf(tw, "%s\t%s\t%s\n", a.ID, a.Name, fmtTime(a.CreatedAt))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(a)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(a)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.StringVar(&name, "name", "", "Pipeline name; leave it empty to generate a random name")
	fs.UintVar(&replicasCount, "replicas", 1, "Pipeline replica size")
	fs.StringVar(&configFile, "config-file", "fluent-bit.conf", "Fluent Bit config file used by pipeline")
	fs.StringVar(&secretsFile, "secrets-file", "", "Optional file where secrets are defined. You can store key values and reference them inside your config like so:\n{{ secrets.foo }}")
	fs.StringVar(&secretsFormat, "secrets-format", "auto", "Secrets file format. Allowed: auto, env, json, yaml. Auto tries to detect it from file extension")
	fs.StringArrayVar(&files, "file", nil, "Optional file. You can reference this file contents from your config like so:\n{{ files.myfile }}\nPass as many as you want; bear in mind the file name can only contain alphanumeric characters.")
	fs.BoolVar(&encryptFiles, "encrypt-files", false, "Encrypt file contents")
	fs.StringVar(&image, "image", "", "Fluent-bit docker image")
	fs.BoolVar(&autoCreatePortsFromConfig, "auto-create-ports", true, "Automatically create pipeline ports from config")
	fs.BoolVar(&skipConfigValidation, "skip-config-validation", false, "Opt-in to skip config validation (Use with caution as this option might be removed soon)")
	fs.StringVar(&resourceProfileName, "resource-profile", cloud.DefaultResourceProfileName, "Resource profile name")
	fs.StringSliceVar(&metadataPairs, "metadata", nil, "Metadata to attach to the pipeline in the form of key:value. You could instead use a file with the --metadata-file option")
	fs.StringVar(&metadataFile, "metadata-file", "", "Metadata JSON file to attach to the pipeline intead of passing multiple --metadata flags")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", config.CompleteCoreInstances)
	_ = cmd.RegisterFlagCompletionFunc("secrets-format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"auto", "env", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("resource-profile", config.CompleteResourceProfiles)

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default core-instance key from config cmd.

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

func parseCreatePipelineSecret(file, format string) ([]cloud.CreatePipelineSecret, error) {
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

	var secrets []cloud.CreatePipelineSecret
	switch format {
	case "env", "dotenv":
		m, err := godotenv.Parse(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloud.CreatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloud.CreatePipelineSecret{
				Key:   k,
				Value: []byte(v),
			})
		}
	case "json":
		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloud.CreatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloud.CreatePipelineSecret{
				Key:   k,
				Value: []byte(fmt.Sprintf("%v", v)),
			})
		}
	case "yml", "yaml":
		var m map[string]interface{}
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("could not parse secrets file %q: %w", file, err)
		}

		secrets = make([]cloud.CreatePipelineSecret, 0, len(m))
		for k, v := range m {
			secrets = append(secrets, cloud.CreatePipelineSecret{
				Key:   k,
				Value: []byte(fmt.Sprintf("%v", v)),
			})
		}
	}

	return secrets, nil
}

func parseMetadataPairs(pairs []string) (*json.RawMessage, error) {
	if len(pairs) == 0 {
		return nil, nil
	}

	m := make(map[string]interface{}, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid metadata: %q", pair)
		}

		key := parts[0]
		var val interface{} = parts[1]

		var dest interface{}
		if err := json.Unmarshal([]byte(val.(string)), &dest); err == nil {
			val = dest
		}

		m[key] = val
	}

	b, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("could not marshal metadata: %w", err)
	}

	metadata := &json.RawMessage{}
	*metadata = b

	return metadata, nil
}
