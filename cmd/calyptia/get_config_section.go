package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/calyptia/api/types"
	"github.com/go-logfmt/logfmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newCmdGetConfigSections(config *config) *cobra.Command {
	var last uint
	var before string
	var outputFormat, goTemplate string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "config_sections", // child of `get`
		Short: "List config sections",
		Long: "List all snipets of config sections,\n" +
			"sorted by creation time in descending order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			params := types.ConfigSectionsParams{}
			if last != 0 {
				params.Last = &last
			}
			if before != "" {
				params.Before = &before
			}
			cc, err := config.cloud.ConfigSections(ctx, config.projectID, params)
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, cc.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(cc)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(cc)
			default:
				return renderConfigSectionsTable(cmd.OutOrStdout(), cc, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` config sections. 0 means no limit")
	fs.StringVar(&before, "before", "", "Only show config sections created before the given cursor")
	fs.BoolVar(&showIDs, "show-ids", false, "Show config section IDs. Only applies when output format is table")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func renderConfigSectionsTable(w io.Writer, cc types.ConfigSections, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	fmt.Fprintln(tw, "KIND\tNAME\tPROPERTIES\tAGE")
	for _, cs := range cc.Items {
		if showIDs {
			_, err := fmt.Fprintf(tw, "%s\t", cs.ID)
			if err != nil {
				return err
			}
		}
		props, err := pairsToLogfmt(cs.Properties, true)
		if err != nil {
			return err
		}

		name := pairsName(cs.Properties)

		_, err = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", cs.Kind, name, props, fmtTime(cs.CreatedAt))
		if err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if cc.EndCursor != nil {
		_, err := fmt.Fprintf(w, "\n\n# Previous page:\n\tcalyptia get config_sections --before %s\n", *cc.EndCursor)
		if err != nil {
			return err
		}
	}

	return nil
}

func pairsToLogfmt(pp types.Pairs, skipName bool) (string, error) {
	var buff bytes.Buffer
	enc := logfmt.NewEncoder(&buff)
	for _, p := range pp {
		if skipName && strings.EqualFold(p.Key, "Name") {
			continue
		}

		err := enc.EncodeKeyval(p.Key, p.Value)
		if err != nil {
			return "", fmt.Errorf("encode property key-val: %w", err)
		}
	}

	enc.Reset()

	return buff.String(), nil
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
