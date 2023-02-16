package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config"
)

func newCmdCreateFleet(config *cfg.Config) *cobra.Command {
	var in types.CreateFleet
	var configFile, configFormat string
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:   "fleet",
		Short: "Create fleet",
		Long:  "Create a new fleet with a shared fluent-bit config in where agents can be attached and share that config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var err error
			in.Config, err = readConfig(configFile, configFormat)
			if err != nil {
				return err
			}

			in.ProjectID = config.ProjectID

			created, err := config.Cloud.CreateFleet(ctx, in)
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, created)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tCREATED-AT")
				fmt.Fprintf(tw, "%s\t%s\n", created.ID, created.CreatedAt.Format(time.RFC822))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&in.Name, "name", "", "Name")
	fs.StringVar(&in.MinFluentBitVersion, "min-fluent-bit-version", "", "Optional minimum fluent-bit version that agents must satisfy to join this fleet")
	fs.StringVar(&configFile, "config-file", "fluent-bit.yaml", "Fluent-bit config file")
	fs.StringVar(&configFormat, "config-format", "", "Optional fluent-bit config format (classic, yaml, json)")
	fs.StringSliceVar(&in.Tags, "tags", nil, "Optional tags for this fleet")
	fs.BoolVar(&in.SkipConfigValidation, "skip-config-validation", false, "Option to skip fluent-bit config validation (not recommended)")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("name")

	_ = cmd.RegisterFlagCompletionFunc("config-format", completeConfigFormat)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

func readConfig(filename, format string) (fluentbitconfig.Config, error) {
	var out fluentbitconfig.Config

	if format == "" || strings.ToLower(format) == "auto" {
		format = strings.TrimPrefix(filepath.Ext(filename), ".")
	}

	b, err := readFile(filename)
	if err != nil {
		return out, err
	}

	return fluentbitconfig.ParseAs(string(b), fluentbitconfig.Format(format))
}

func completeConfigFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"yaml", "json", "classic"}, cobra.ShellCompDirectiveNoFileComp
}
