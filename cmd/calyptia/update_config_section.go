package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/completer"
	cfg "github.com/calyptia/cli/pkg/config"
)

func newCmdUpdateConfigSection(config *cfg.Config) *cobra.Command {
	var propsSlice []string
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "config_section CONFIG_SECTION", // child of `update`
		Short:             "Update a config section",
		Long:              "Update a config section either by the plugin kind:name or by its ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteConfigSections,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			configSectionKey := args[0]
			configSectionID, err := completer.LoadConfigSectionID(ctx, configSectionKey)
			if err != nil {
				return fmt.Errorf("load config section ID from key: %w", err)
			}

			cs, err := config.Cloud.ConfigSection(ctx, configSectionID)
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			props := propsFromSlice(propsSlice)
			props = append(types.Pairs{
				{Key: "name", Value: pairsName(cs.Properties)},
			}, props...)

			updated, err := config.Cloud.UpdateConfigSection(ctx, configSectionID, types.UpdateConfigSection{
				Properties: &props,
			})
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, updated)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(updated)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(updated)
			default:
				return renderUpdatedTable(cmd.OutOrStdout(), updated.UpdatedAt)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringSliceVarP(&propsSlice, "prop", "p", nil, "Additional properties; follow the format -p foo=bar -p baz=qux")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("prop", completer.CompletePluginProps)

	return cmd
}
