package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/calyptia/cloud"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func newCmdGetPipelines(config *config) *cobra.Command {
	var format string
	var aggregatorKey string
	var last uint64
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Display latest pipelines from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorID := aggregatorKey
			{
				aa, err := config.fetchAllAggregators()
				if err != nil {
					return err
				}

				a, ok := findAggregatorByName(aa, aggregatorKey)
				if !ok && !validUUID(aggregatorID) {
					return fmt.Errorf("could not find aggregator %q", aggregatorKey)
				}

				if ok {
					aggregatorID = a.ID
				}
			}

			pp, err := config.cloud.AggregatorPipelines(config.ctx, aggregatorID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"Name", "Replicas", "Status", "Age"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, p := range pp {
					tw.AppendRow(table.Row{p.Name, p.ReplicasCount, p.Status.Status, fmtAgo(p.CreatedAt)})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your pipelines: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&aggregatorKey, "aggregator", "", "Parent aggregator ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("aggregator", config.completeAggregators)

	_ = cmd.MarkFlagRequired("aggregator") // TODO: use default aggregator ID from config cmd.

	return cmd
}

func (config *config) fetchAllPipelines() ([]cloud.AggregatorPipeline, error) {
	pp, err := config.cloud.Projects(config.ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("could not prefetch pipelines: %w", err)
	}

	if len(pp) == 0 {
		return nil, nil
	}

	var pipelines []cloud.AggregatorPipeline
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, p := range pp {
		p := p
		g.Go(func() error {
			aa, err := config.cloud.Aggregators(gctx, p.ID, 0)
			if err != nil {
				return err
			}

			g2, gctx2 := errgroup.WithContext(gctx)
			for _, a := range aa {
				a := a
				g2.Go(func() error {
					got, err := config.cloud.AggregatorPipelines(gctx2, a.ID, 0)
					if err != nil {
						return err
					}

					mu.Lock()
					pipelines = append(pipelines, got...)
					mu.Unlock()

					return nil
				})
			}
			return g2.Wait()
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return pipelines, nil
}

func (config *config) completePipelines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.fetchAllPipelines()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return pipelinesKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

// pipelinesKeys returns unique pipeline names first and then IDs.
func pipelinesKeys(aa []cloud.AggregatorPipeline) []string {
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

func findPipelineByName(pp []cloud.AggregatorPipeline, name string) (cloud.AggregatorPipeline, bool) {
	for _, pip := range pp {
		if pip.Name == name {
			return pip, true
		}
	}
	return cloud.AggregatorPipeline{}, false
}
