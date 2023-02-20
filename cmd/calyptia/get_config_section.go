package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/go-logfmt/logfmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func newCmdGetConfigSections(config *cfg.Config) *cobra.Command {
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
			cc, err := config.Cloud.ConfigSections(ctx, config.ProjectID, params)
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

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

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

func pairsName(pp types.Pairs) string {
	if v, ok := pp.Get("Name"); ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
