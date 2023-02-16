package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdGetPipelineSecrets(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_secrets",
		Short: "Get pipeline secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ss, err := config.Cloud.PipelineSecrets(config.Ctx, pipelineID, cloud.PipelineSecretsParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline secrets: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, ss.Items)
			}

			switch outputFormat {
			case "table":
				renderPipelineSecrets(cmd.OutOrStdout(), ss.Items, showIDs)
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline secrets. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineSecrets(w io.Writer, ss []cloud.PipelineSecret, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "KEY\tAGE")
	for _, s := range ss {
		if showIDs {
			fmt.Fprintf(tw, "%s\t", s.ID)
		}
		fmt.Fprintf(tw, "%s\t%s\n", s.Key, fmtTime(s.CreatedAt))
	}
	tw.Flush()
}
