package members

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"

	cfg "github.com/chronosphereio/calyptia-cli/config"
	"github.com/chronosphereio/calyptia-cli/formatters"
)

func NewCmdGetMembers(config *cfg.Config) *cobra.Command {
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "members",
		Short: "Display latest members from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			mm, err := config.Cloud.Members(config.Ctx, config.ProjectID, cloud.MembersParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your project members: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, mm.Items)
			}

			switch outputFormat {
			case "table":
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
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(mm.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(mm.Items)
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

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
