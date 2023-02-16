package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdCreatePipelineFile(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var file string
	var encrypt bool
	var outputFormat, goTemplate string
	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Create a new file within a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := filepath.Base(file)
			name = strings.TrimSuffix(name, filepath.Ext(name))

			contents, err := readFile(file)
			if err != nil {
				return err
			}

			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			out, err := config.Cloud.CreatePipelineFile(config.Ctx, pipelineID, cloud.CreatePipelineFile{
				Name:      name,
				Contents:  contents,
				Encrypted: encrypt,
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, out)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", out.ID, fmtTime(out.CreatedAt))
				tw.Flush()

				return nil
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Pipeline ID or name")
	fs.StringVar(&file, "file", "", "File path. You will be able to reference the file from a fluentbit config using its base name without the extension. Ex: `some_dir/my_file.txt` will be referenced as `{{files.my_file}}`")
	fs.BoolVar(&encrypt, "encrypt", false, "Encrypt file contents")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("pipeline")
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
