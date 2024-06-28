package environment

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetEnvironment(cfg *config.Config) *cobra.Command {
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "environment",
		Short: "Get environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			ee, err := cfg.Cloud.Environments(ctx, cfg.ProjectID, cloudtypes.EnvironmentsParams{Last: &last})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ee)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ee.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ee.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 3, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprint(tw, "NAME\t")
				fmt.Fprintln(tw, "AGE")
				for _, m := range ee.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", m.ID)
					}

					fmt.Fprintf(tw, "%s\t", m.Name)
					fmt.Fprintln(tw, formatters.FmtTime(m.CreatedAt))
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` members. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include member IDs in table output")
	formatters.BindFormatFlags(cmd)

	return cmd
}
