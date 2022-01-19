package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdGetAggregators(config *config) *cobra.Command {
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "aggregators",
		Short: "Display latest aggregators from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			aa, err := config.cloud.Aggregators(config.ctx, config.projectID, cloud.LastAggregators(last))
			if err != nil {
				return fmt.Errorf("could not fetch your aggregators: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tAGE")
				for _, a := range aa {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\n", a.Name, fmtAgo(a.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(aa)
				if err != nil {
					return fmt.Errorf("could not json encode your aggregators: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` aggregators. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include aggregator IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func (config *config) completeAggregators(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.Aggregators(config.ctx, config.projectID)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if aa == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return aggregatorsKeys(aa), cobra.ShellCompDirectiveNoFileComp
}

// aggregatorsKeys returns unique aggregator names first and then IDs.
func aggregatorsKeys(aa []cloud.Aggregator) []string {
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

func (config *config) loadAggregatorID(aggregatorKey string) (string, error) {
	aa, err := config.cloud.Aggregators(config.ctx, config.projectID, cloud.AggregatorsWithName(aggregatorKey), cloud.LastAggregators(2))
	if err != nil {
		return "", err
	}

	if len(aa) != 1 && !validUUID(aggregatorKey) {
		if len(aa) != 0 {
			return "", fmt.Errorf("ambiguous aggregator name %q, use ID instead", aggregatorKey)
		}

		return "", fmt.Errorf("could not find aggregator %q", aggregatorKey)
	}

	if len(aa) == 1 {
		return aa[0].ID, nil
	}

	return aggregatorKey, nil

}
