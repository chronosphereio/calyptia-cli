package clusterobject

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

func NewCmdGetClusterObjects(cfg *config.Config) *cobra.Command {
	var coreInstanceKey string
	var last uint
	var environment string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "cluster_objects",
		Short: "Get cluster objects",
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

			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			out, err := cfg.Cloud.ClusterObjects(ctx, coreInstanceID, cloudtypes.ClusterObjectParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your cluster objects: %w", err)
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
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tKIND\tCREATED AT")
				for _, c := range out.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", c.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", c.Name, string(c.Kind), formatters.FmtTime(c.CreatedAt))
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Core Instance to list cluster objects from")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` cluster objects. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("core-instance")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	return cmd
}
