package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newCmdGetPipelines(config *config) *cobra.Command {
	var format string
	var aggregatorKey string
	var last uint64
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Display latest pipelines from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorID, err := config.loadAggregatorID(aggregatorKey)
			if err != nil {
				return err
			}

			pp, err := config.cloud.AggregatorPipelines(config.ctx, aggregatorID, cloud.LastPipelines(last))
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tAGE")
				for _, p := range pp {
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", p.Name, p.ReplicasCount, p.Status.Status, fmtAgo(p.CreatedAt))
				}
				tw.Flush()
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

func newCmdGetPipeline(config *config) *cobra.Command {
	var format string
	var lastEndpoints, lastConfigHistory uint64
	var includeEndpoints, includeConfigHistory bool
	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		Short:             "Display a pipelines by ID or name",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			var pip cloud.AggregatorPipeline
			var ports []cloud.PipelinePort
			var history []cloud.PipelineConfig
			if format == "table" && (includeEndpoints || includeConfigHistory) {
				g, gctx := errgroup.WithContext(config.ctx)
				g.Go(func() error {
					var err error
					pip, err = config.cloud.AggregatorPipeline(config.ctx, pipelineID)
					if err != nil {
						return fmt.Errorf("could not fetch your pipeline: %w", err)
					}
					return nil
				})
				if includeEndpoints {
					g.Go(func() error {
						var err error
						ports, err = config.cloud.PipelinePorts(gctx, pipelineID, lastEndpoints)
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline endpoints: %w", err)
						}
						return nil
					})
				}
				if includeConfigHistory {
					g.Go(func() error {
						var err error
						history, err = config.cloud.PipelineConfigHistory(gctx, pipelineID, lastConfigHistory)
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline config history: %w", err)
						}
						return nil
					})
				}

				if err := g.Wait(); err != nil {
					return err
				}
			} else {
				var err error
				pip, err = config.cloud.AggregatorPipeline(config.ctx, pipelineID)
				if err != nil {
					return fmt.Errorf("could not fetch your pipeline: %w", err)
				}
			}

			switch format {
			case "table":
				{
					tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
					fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tAGE")
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", pip.Name, pip.ReplicasCount, pip.Status.Status, fmtAgo(pip.CreatedAt))
					tw.Flush()
				}
				if includeEndpoints {
					fmt.Println()
					renderEndpointsTable(os.Stdout, ports)
				}
				if includeConfigHistory {
					fmt.Println()
					renderPipelineConfigHistory(os.Stdout, history)
				}
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pip)
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
	fs.BoolVar(&includeEndpoints, "include-endpoints", false, "Include endpoints in output (only available with table format)")
	fs.BoolVar(&includeConfigHistory, "include-config-history", false, "Include config history in output (only available with table format)")
	fs.Uint64Var(&lastEndpoints, "last-endpoints", 0, "Last `N` pipeline endpoints if included. 0 means no limit")
	fs.Uint64Var(&lastConfigHistory, "last-config-history", 0, "Last `N` pipeline config history if included. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func (config *config) fetchAllPipelines() ([]cloud.AggregatorPipeline, error) {
	pp, err := config.cloud.Projects(config.ctx)
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
			aa, err := config.cloud.Aggregators(gctx, p.ID)
			if err != nil {
				return err
			}

			if len(aa) == 0 {
				return nil
			}

			g2, gctx2 := errgroup.WithContext(gctx)
			for _, a := range aa {
				a := a
				g2.Go(func() error {
					got, err := config.cloud.AggregatorPipelines(gctx2, a.ID)
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

	var uniquePipelines []cloud.AggregatorPipeline
	pipelineIDs := map[string]struct{}{}
	for _, pip := range pipelines {
		if _, ok := pipelineIDs[pip.ID]; !ok {
			uniquePipelines = append(uniquePipelines, pip)
			pipelineIDs[pip.ID] = struct{}{}
		}
	}

	return uniquePipelines, nil
}

func (config *config) fetchAllProjectPipelines(projectID string) ([]cloud.AggregatorPipeline, error) {
	aa, err := config.cloud.Aggregators(config.ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("could not prefetch aggregators: %w", err)
	}

	if len(aa) == 0 {
		return nil, nil
	}

	var pipelines []cloud.AggregatorPipeline
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.ctx)
	for _, a := range aa {
		a := a
		g.Go(func() error {
			got, err := config.cloud.AggregatorPipelines(gctx, a.ID)
			if err != nil {
				return err
			}

			mu.Lock()
			pipelines = append(pipelines, got...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var uniquePipelines []cloud.AggregatorPipeline
	pipelineIDs := map[string]struct{}{}
	for _, pip := range pipelines {
		if _, ok := pipelineIDs[pip.ID]; !ok {
			uniquePipelines = append(uniquePipelines, pip)
			pipelineIDs[pip.ID] = struct{}{}
		}
	}

	return uniquePipelines, nil
}

func (config *config) completePipelines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var pp []cloud.AggregatorPipeline
	if config.defaultProject != "" {
		var err error
		pp, err = config.fetchAllProjectPipelines(config.defaultProject)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
	} else {
		var err error
		pp, err = config.fetchAllPipelines()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
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

func (config *config) loadPipelineID(pipelineKey string) (string, error) {
	if config.defaultProject != "" {
		var err error
		pp, err := config.cloud.ProjectPipelines(config.ctx, config.defaultProject, cloud.PipelinesWithName(pipelineKey), cloud.LastPipelines(2))
		if err != nil {
			return "", err
		}

		if len(pp) != 1 && !validUUID(pipelineKey) {
			if len(pp) != 0 {
				return "", fmt.Errorf("ambiguous pipeline name %q, use ID instead", pipelineKey)
			}

			return "", fmt.Errorf("could not find pipeline %q", pipelineKey)
		}

		if len(pp) == 1 {
			return pp[0].ID, nil
		}

		return pipelineKey, nil
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
			pp, err := config.cloud.ProjectPipelines(gctx, proj.ID, cloud.PipelinesWithName(pipelineKey), cloud.LastPipelines(2))
			if err != nil {
				return err
			}

			if len(pp) != 1 && !validUUID(pipelineKey) {
				if len(pp) != 0 {
					return fmt.Errorf("ambiguous pipeline name %q, use ID instead", pipelineKey)
				}

				return fmt.Errorf("could not find pipeline %q", pipelineKey)
			}

			if len(pp) == 1 {
				mu.Lock()
				founds = append(founds, pp[0].ID)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return "", err
	}

	if len(founds) != 1 && !validUUID(pipelineKey) {
		if len(founds) != 0 {
			return "", fmt.Errorf("ambiguous pipeline name %q, use ID instead", pipelineKey)
		}

		return "", fmt.Errorf("could not find pipeline %q", pipelineKey)
	}

	if len(founds) == 1 {
		return founds[0], nil
	}

	return pipelineKey, nil
}
