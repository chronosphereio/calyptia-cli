package members

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

func NewCmdGetMembers(cfg *config.Config) *cobra.Command {
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "members",
		Short: "Display latest members from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			mm, err := cfg.Cloud.Members(ctx, cfg.ProjectID, cloudtypes.MembersParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your project members: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), mm.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(mm.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(mm.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprint(tw, "EMAIL\tNAME\tROLES\tPERMISSIONS\t")
				if showIDs {
					fmt.Fprint(tw, "MEMBER-ID\t")
				}
				fmt.Fprintln(tw, "AGE")
				for _, m := range mm.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", m.User.ID)
					}
					roles := make([]string, len(m.Roles))
					for i, r := range m.Roles {
						roles[i] = string(r)
					}
					permissions := strings.Join(m.Permissions, ", ")
					if permissions == "" {
						permissions = "all"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t", m.User.Email, m.User.Name, strings.Join(roles, ", "), permissions)
					if showIDs {
						fmt.Fprintf(tw, "%s\t", m.ID)
					}
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
