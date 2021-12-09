package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newCmdGetAggregators(config *config) *cobra.Command {
	var format string
	var projectKey string
	var last uint64
	cmd := &cobra.Command{
		Use:   "aggregators",
		Short: "Display latest aggregators from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectKey == "" {
				return errors.New("project required")
			}

			projectID, err := config.loadProjectID(projectKey)
			if err != nil {
				return err
			}

			aa, err := config.cloud.Aggregators(config.ctx, projectID, cloud.LastAggregators(last))
			if err != nil {
				return fmt.Errorf("could not fetch your aggregators: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "NAME\tAGE")
				for _, a := range aa {
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
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&projectKey, "project", config.defaultProject, "Parent project ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` aggregators. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("project", config.completeProjects)

	return cmd
}

func (config *config) fetchAllAggregators() ([]cloud.Aggregator, error) {
	pp, err := config.cloud.Projects(config.ctx)
	if err != nil {
		return nil, fmt.Errorf("could not prefetch projects: %w", err)
	}

	if len(pp) == 0 {
		return nil, nil
	}

	var aa []cloud.Aggregator
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, p := range pp {
		p := p
		g.Go(func() error {
			got, err := config.cloud.Aggregators(gctx, p.ID)
			if err != nil {
				return fmt.Errorf("could not fetch aggregators from project: %w", err)
			}

			mu.Lock()
			aa = append(aa, got...)
			mu.Unlock()

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("could not fetch projects aggregators: %w", err)
	}

	var uniqueAggregators []cloud.Aggregator
	aggregatorsIDs := map[string]struct{}{}
	for _, a := range aa {
		if _, ok := aggregatorsIDs[a.ID]; !ok {
			uniqueAggregators = append(uniqueAggregators, a)
			aggregatorsIDs[a.ID] = struct{}{}
		}
	}

	return uniqueAggregators, nil
}

func (config *config) completeAggregators(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var aa []cloud.Aggregator
	if config.defaultProject != "" {
		var err error
		aa, err = config.cloud.Aggregators(config.ctx, config.defaultProject)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
	} else {
		var err error
		aa, err = config.fetchAllAggregators()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
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
	if config.defaultProject != "" {
		var err error
		aa, err := config.cloud.Aggregators(config.ctx, config.defaultProject, cloud.AggregatorsWithName(aggregatorKey), cloud.LastAggregators(2))
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

	projs, err := config.cloud.Projects(config.ctx)
	if err != nil {
		return "", err
	}

	var founds []string
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, proj := range projs {
		proj := proj
		g.Go(func() error {
			aa, err := config.cloud.Aggregators(gctx, proj.ID, cloud.AggregatorsWithName(aggregatorKey), cloud.LastAggregators(2))
			if err != nil {
				return err
			}

			if len(aa) != 1 && !validUUID(aggregatorKey) {
				if len(aa) != 0 {
					return fmt.Errorf("ambiguous aggregator name %q, use ID instead", aggregatorKey)
				}

				return fmt.Errorf("could not find aggregator %q", aggregatorKey)
			}

			if len(aa) == 1 {
				mu.Lock()
				founds = append(founds, aa[0].ID)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	if len(founds) != 1 && !validUUID(aggregatorKey) {
		if len(founds) != 0 {
			return "", fmt.Errorf("ambiguous aggregator name %q, use ID instead", aggregatorKey)
		}

		return "", fmt.Errorf("could not find aggregator %q", aggregatorKey)
	}

	if len(founds) == 1 {
		return founds[0], nil
	}

	return aggregatorKey, nil
}
