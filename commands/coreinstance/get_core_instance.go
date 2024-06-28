package coreinstance

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetCoreInstances(cfg *config.Config) *cobra.Command {
	var last uint
	var showIDs bool
	var showMetadata bool
	var environment string

	cmd := &cobra.Command{
		Use:     "core_instances",
		Aliases: []string{"instances", "core_instances"},
		Short:   "Display latest core instances from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}
			var params cloudtypes.CoreInstancesParams

			params.Last = &last
			if environmentID != "" {
				params.EnvironmentID = &environmentID
			}

			out, err := cfg.Cloud.CoreInstances(ctx, cfg.ProjectID, params)
			if err != nil {
				return fmt.Errorf("could not fetch your core instances: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), out)
			}

			switch outputFormat {
			case formatters.OutputFormatJSON:
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out.Items)
			case formatters.OutputFormatYAML:
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprint(tw, "NAME\tVERSION\tENVIRONMENT\tPIPELINES\tTAGS\tSTATUS\tAGE")
				if showMetadata {
					fmt.Fprintln(tw, "\tMETADATA")
				} else {
					fmt.Fprintln(tw, "")
				}
				for _, a := range out.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s", a.Name, a.Version, a.EnvironmentName, a.PipelinesCount, strings.Join(a.Tags, ","), a.Status, formatters.FmtTime(a.CreatedAt))
					if showMetadata {
						metadata, err := formatters.FilterOutEmptyMetadata(a.Metadata)
						if err != nil {
							continue
						}
						fmt.Fprintf(tw, "\t%s\n", string(metadata))
					} else {
						fmt.Fprintln(tw, "")
					}
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` core instances. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include core instance IDs in table output")
	fs.BoolVar(&showMetadata, "show-metadata", false, "Include core instance metadata in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name.")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)

	return cmd
}
