package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetAggregators(config *config) *cobra.Command {
	var last uint64
	var format string
	var showIDs bool
	var showMetadata bool
	var environmentKey string

	cmd := &cobra.Command{
		Use:     "core_instances",
		Aliases: []string{"instances", "aggregators"},
		Short:   "Display latest core instances from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environmentKey != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environmentKey)
				if err != nil {
					return err
				}
			}
			var params cloud.AggregatorsParams

			params.Last = &last
			if environmentID != "" {
				params.EnvironmentID = &environmentID
			}

			aa, err := config.cloud.Aggregators(config.ctx, config.projectID, params)
			if err != nil {
				return fmt.Errorf("could not fetch your core instances: %w", err)
			}

			switch format {
			case "table":
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
				for _, a := range aa.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s", a.Name, a.Version, a.EnvironmentName, a.PipelinesCount, strings.Join(a.Tags, ","), a.Status, fmtAgo(a.CreatedAt))
					if showMetadata && a.Metadata != nil {
						filterOutEmptyMetadata(a.Metadata)
						fmt.Fprintf(tw, "\t%s\n", *a.Metadata)
					} else {
						fmt.Fprintln(tw, "")
					}
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(aa.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your core instances: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` core instances. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include core instance IDs in table output")
	fs.BoolVar(&showMetadata, "show-metadata", false, "Include core instance metadata in table output")
	fs.StringVar(&environmentKey, "environment", "", "Calyptia environment name or ID")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func (config *config) completeAggregators(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.Aggregators(config.ctx, config.projectID, cloud.AggregatorsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return aggregatorsKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
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

func (config *config) loadAggregatorID(aggregatorKey string, environmentID string) (string, error) {
	params := cloud.AggregatorsParams{
		Name: &aggregatorKey,
		Last: ptr(uint64(2)),
	}

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := config.cloud.Aggregators(config.ctx, config.projectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !validUUID(aggregatorKey) {
		if len(aa.Items) != 0 {
			return "", fmt.Errorf("ambiguous core instance name %q, use ID instead", aggregatorKey)
		}

		return "", fmt.Errorf("could not find core instance %q", aggregatorKey)
	}

	if len(aa.Items) == 1 {
		return aa.Items[0].ID, nil
	}

	return aggregatorKey, nil
}
