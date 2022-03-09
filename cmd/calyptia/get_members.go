package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetMembers(config *config) *cobra.Command {
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Display latest members from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			mm, err := config.cloud.Members(config.ctx, config.projectID, cloud.MembersParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your project members: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprint(tw, "EMAIL\tNAME\tROLES\t")
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
					fmt.Fprintf(tw, "%s\t%s\t%s\t", m.User.Email, m.User.Name, strings.Join(roles, ", "))
					if showIDs {
						fmt.Fprintf(tw, "%s\t", m.ID)
					}
					fmt.Fprintln(tw, fmtAgo(m.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(mm.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your project members: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` members. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include member IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}
