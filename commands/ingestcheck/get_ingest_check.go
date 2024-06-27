package ingestcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetIngestCheck(c *cfg.Config) *cobra.Command {
	var (
		outputFormat string
		showIDs      bool
		goTemplate   string
	)
	cmd := &cobra.Command{
		Use:   "ingest_check INGEST_CHECK_ID",
		Short: "Get a specific ingest check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]
			check, err := c.Cloud.IngestCheck(ctx, id)
			if err != nil {
				return err
			}
			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, check)
			}
			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 3, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprint(tw, "STATUS\tRETRIES\t")
				fmt.Fprintln(tw, "AGE")
				if showIDs {
					fmt.Fprintf(tw, "%s\t", check.ID)
				}

				fmt.Fprintf(tw, "%s\t", check.Status)
				fmt.Fprintf(tw, "%d\t", check.Retries)
				fmt.Fprintln(tw, formatters.FmtTime(check.CreatedAt))
				err := tw.Flush()
				if err != nil {
					return err
				}
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(check)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(check)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}

			return nil
		},
	}
	fs := cmd.Flags()
	fs.BoolVar(&showIDs, "show-ids", false, "Include member IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	return cmd
}

func NewCmdGetIngestChecks(c *cfg.Config) *cobra.Command {
	var (
		outputFormat string
		showIDs      bool
		last         uint
		goTemplate   string
		environment  string
	)
	completer := completer.Completer{Config: c}

	cmd := &cobra.Command{
		Use:   "ingest_checks CORE_INSTANCE",
		Short: "Get a list of ingest checks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			aggregatorID, err := completer.LoadCoreInstanceID(id, environmentID)
			if err != nil {
				return err
			}
			check, err := c.Cloud.IngestChecks(ctx, aggregatorID, types.IngestChecksParams{Last: &last})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, check.Items)
			}
			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 3, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprint(tw, "STATUS\tRETRIES\t")
				fmt.Fprintln(tw, "AGE")
				for _, m := range check.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", m.ID)
					}

					fmt.Fprintf(tw, "%s\t", m.Status)
					fmt.Fprintf(tw, "%d\t", m.Retries)
					fmt.Fprintln(tw, formatters.FmtTime(m.CreatedAt))
				}
				err := tw.Flush()
				if err != nil {
					return err
				}
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(check.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(check.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}

			return nil
		},
	}
	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` members. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include member IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")
	fs.StringVar(&environment, "environment", "default", "Environment name")
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	return cmd
}
