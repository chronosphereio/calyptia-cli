package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/calyptia/api/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newCmdUpdateConfigSection(config *config) *cobra.Command {
	var propsSlice []string
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:               "config_section CONFIG_SECTION", // child of `update`
		Short:             "Update a config section",
		Long:              "Update a config section either by the plugin kind:name or by its ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeConfigSections,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			configSectionKey := args[0]
			configSectionID, err := config.loadConfigSectionID(ctx, configSectionKey)
			if err != nil {
				return fmt.Errorf("load config section ID from key: %w", err)
			}

			cs, err := config.cloud.ConfigSection(ctx, configSectionID)
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			props := propsFromSlice(propsSlice)
			props = append(types.Pairs{
				{Key: "name", Value: pairsName(cs.Properties)},
			}, props...)

			updated, err := config.cloud.UpdateConfigSection(ctx, configSectionID, types.UpdateConfigSection{
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

	_ = cmd.RegisterFlagCompletionFunc("prop", config.completePluginProps)

	return cmd
}

func (config *config) loadConfigSectionID(ctx context.Context, key string) (string, error) {
	cc, err := config.cloud.ConfigSections(ctx, config.projectID, types.ConfigSectionsParams{})
	if err != nil {
		return "", fmt.Errorf("cloud: %w", err)
	}

	if len(cc.Items) == 0 {
		return "", errors.New("cloud: no config sections yet")
	}

	for _, cs := range cc.Items {
		if key == cs.ID {
			return cs.ID, nil
		}
	}

	var foundID string
	var foundCount uint

	for _, cs := range cc.Items {
		kindName := configSectionKindName(cs)
		if kindName == key {
			foundID = cs.ID
			foundCount++
		}
	}

	if foundCount > 1 {
		return "", fmt.Errorf("ambiguous config section %q, try using the ID", key)
	}

	if foundCount == 0 {
		return "", fmt.Errorf("could not find config section with key %q", key)
	}

	return foundID, nil
}

func (config *config) completeConfigSections(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	cc, err := config.cloud.ConfigSections(ctx, config.projectID, types.ConfigSectionsParams{})
	if err != nil {
		cobra.CompErrorln(fmt.Sprintf("cloud: %v", err))
		return nil, cobra.ShellCompDirectiveError
	}

	if len(cc.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return configSectionKeys(cc.Items), cobra.ShellCompDirectiveNoFileComp
}

func configSectionKeys(cc []types.ConfigSection) []string {
	kindNameCounts := map[string]uint{}
	for _, cs := range cc {
		kindName := configSectionKindName(cs)
		if _, ok := kindNameCounts[kindName]; ok {
			kindNameCounts[kindName]++
			continue
		}

		kindNameCounts[kindName] = 1
	}

	var out []string
	for _, cs := range cc {
		kindName := configSectionKindName(cs)
		if count, ok := kindNameCounts[kindName]; ok && count == 1 {
			out = append(out, kindName)
		} else {
			out = append(out, cs.ID)
		}
	}

	return out
}

func configSectionKindName(cs types.ConfigSection) string {
	return fmt.Sprintf("%s:%s", cs.Kind, pairsName(cs.Properties))
}

func pairsName(pp types.Pairs) string {
	if v, ok := pp.Get("Name"); ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
