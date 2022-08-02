package main

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func (config *config) completeEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.Environments(config.ctx, config.projectID, cloud.EnvironmentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return environmentNames(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

// environmentNames returns unique environment names that belongs to a project.
func environmentNames(aa []cloud.Environment) []string {
	var out []string
	for _, a := range aa {
		out = append(out, a.Name)
	}
	return out
}

func (config *config) loadEnvironmentID(environmentName string) (string, error) {
	aa, err := config.cloud.Environments(config.ctx, config.projectID, cloud.EnvironmentsParams{
		Name: &environmentName,
		Last: ptr(uint64(1)),
	})
	if err != nil {
		return "", err
	}

	if len(aa.Items) == 0 {
		return "", fmt.Errorf("could not find environment %q", environmentName)

	}

	return aa.Items[0].ID, nil
}

func newCmdGetEnvironment(c *config) *cobra.Command {
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "environment",
		Short: "Get environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			environments, err := c.cloud.Environments(ctx, c.projectID, cloud.EnvironmentsParams{Last: &last})
			if err != nil {
				return err
			}
			if err != nil {
				return fmt.Errorf("could not fetch your project members: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 3, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprint(tw, "NAME\t")
				fmt.Fprintln(tw, "AGE")
				for _, m := range environments.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", m.ID)
					}

					fmt.Fprintf(tw, "%s\t", m.Name)
					fmt.Fprintln(tw, fmtAgo(m.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(environments.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your environments: %w", err)
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
	return cmd

}
