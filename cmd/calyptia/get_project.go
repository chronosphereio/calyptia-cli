package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdGetProjects(config *config) *cobra.Command {
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "Display latest projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			pp, err := config.cloud.Projects(config.ctx, cloud.LastProjects(last))
			if err != nil {
				return fmt.Errorf("could not fetch your projects: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tAGE")
				for _, p := range pp {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", p.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\n", p.Name, fmtAgo(p.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your projects: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` projects. 0 means no limit")
	fs.StringVarP(&format, "output-format", "f", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include project IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func (config *config) completeProjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.cloud.Projects(config.ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(pp) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return projectsKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

// projectsKeys returns unique project names first and then IDs.
func projectsKeys(aa []cloud.Project) []string {
	namesCount := map[string]int{}
	for _, a := range aa {
		if _, ok := namesCount[a.Name]; ok {
			namesCount[a.Name] += 1
			continue
		}

		namesCount[a.Name] = 1
	}

	var out []string

	for _, a := range aa {
		var nameIsUnique bool
		for name, count := range namesCount {
			if a.Name == name && count == 1 {
				nameIsUnique = true
				break
			}
		}
		if nameIsUnique {
			out = append(out, a.Name)
			continue
		}

		out = append(out, a.ID)
	}

	return out
}

func (config *config) loadProjectID(projectKey string) (string, error) {
	pp, err := config.cloud.Projects(config.ctx, cloud.ProjectsWithName(projectKey), cloud.LastProjects(2))
	if err != nil {
		return "", err
	}

	if len(pp) != 1 && !validUUID(projectKey) {
		if len(pp) != 0 {
			return "", fmt.Errorf("ambiguous project name %q, use ID instead", projectKey)
		}

		return "", fmt.Errorf("could not find project %q", projectKey)
	}

	if len(pp) == 1 {
		return pp[0].ID, nil
	}

	return projectKey, nil
}
