package ingestcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetIngestCheck(cfg *config.Config) *cobra.Command {
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "ingest_check INGEST_CHECK_ID",
		Short: "Get a specific ingest check",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			id := args[0]
			check, err := cfg.Cloud.IngestCheck(ctx, id)
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), check)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(check)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(check)
			default:
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
				return tw.Flush()
			}
		},
	}
	fs := cmd.Flags()
	fs.BoolVar(&showIDs, "show-ids", false, "Include member IDs in table output")
	formatters.BindFormatFlags(cmd)
	return cmd
}

func NewCmdGetIngestChecks(cfg *config.Config) *cobra.Command {
	var (
		showIDs     bool
		last        uint
		environment string
	)

	cmd := &cobra.Command{
		Use:   "ingest_checks CORE_INSTANCE",
		Short: "Get a list of ingest checks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			id := args[0]
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}
			aggregatorID, err := cfg.Completer.LoadCoreInstanceID(ctx, id, environmentID)
			if err != nil {
				return err
			}
			check, err := cfg.Cloud.IngestChecks(ctx, aggregatorID, cloudtypes.IngestChecksParams{Last: &last})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), check.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(check.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(check.Items)
			default:
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
				return tw.Flush()
			}
		},
	}
	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` members. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include member IDs in table output")
	fs.StringVar(&environment, "environment", "default", "Environment name")
	formatters.BindFormatFlags(cmd)
	return cmd
}
