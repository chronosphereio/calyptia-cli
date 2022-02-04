package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newCmdGetPipelines(config *config) *cobra.Command {
	var aggregatorKey string
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Display latest pipelines from an aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorID, err := config.loadAggregatorID(aggregatorKey)
			if err != nil {
				return err
			}

			pp, err := config.cloud.Pipelines(config.ctx, aggregatorID, cloud.PipelinesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tAGE")
				for _, p := range pp {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", p.ID)
					}
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
	fs.StringVar(&aggregatorKey, "aggregator", "", "Parent aggregator ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include pipeline IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("aggregator", config.completeAggregators)

	_ = cmd.MarkFlagRequired("aggregator") // TODO: use default aggregator ID from config cmd.

	return cmd
}

func newCmdGetPipeline(config *config) *cobra.Command {
	var onlyConfig bool
	var lastEndpoints, lastConfigHistory, lastSecrets uint64
	var includeEndpoints, includeConfigHistory, includeSecrets bool
	var showIDs bool
	var format string
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

			var pip cloud.Pipeline
			var ports []cloud.PipelinePort
			var history []cloud.PipelineConfig
			var secrets []cloud.PipelineSecret
			if format == "table" && (includeEndpoints || includeConfigHistory) {
				g, gctx := errgroup.WithContext(config.ctx)
				g.Go(func() error {
					var err error
					pip, err = config.cloud.Pipeline(config.ctx, pipelineID)
					if err != nil {
						return fmt.Errorf("could not fetch your pipeline: %w", err)
					}
					return nil
				})
				if includeEndpoints {
					g.Go(func() error {
						var err error
						ports, err = config.cloud.PipelinePorts(gctx, pipelineID, cloud.PipelinePortsParams{
							Last: &lastEndpoints,
						})
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline endpoints: %w", err)
						}
						return nil
					})
				}
				if includeConfigHistory {
					g.Go(func() error {
						var err error
						history, err = config.cloud.PipelineConfigHistory(gctx, pipelineID, cloud.PipelineConfigHistoryParams{
							Last: &lastConfigHistory,
						})
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline config history: %w", err)
						}
						return nil
					})
				}
				if includeSecrets {
					g.Go(func() error {
						var err error
						secrets, err = config.cloud.PipelineSecrets(gctx, pipelineID, cloud.PipelineSecretsParams{
							Last: &lastSecrets,
						})
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline secrets: %w", err)
						}
						return nil
					})
				}

				if err := g.Wait(); err != nil {
					return err
				}
			} else {
				var err error
				pip, err = config.cloud.Pipeline(config.ctx, pipelineID)
				if err != nil {
					return fmt.Errorf("could not fetch your pipeline: %w", err)
				}
			}

			if onlyConfig {
				fmt.Println(strings.TrimSpace(pip.Config.RawConfig))
				return nil
			}

			switch format {
			case "table":
				{
					tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
					if showIDs {
						fmt.Fprint(tw, "ID\t")
					}
					fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tAGE")
					if showIDs {
						fmt.Fprintf(tw, "%s\t", pip.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", pip.Name, pip.ReplicasCount, pip.Status.Status, fmtAgo(pip.CreatedAt))
					tw.Flush()
				}
				if includeEndpoints {
					fmt.Println()
					renderEndpointsTable(os.Stdout, ports, showIDs)
				}
				if includeConfigHistory {
					fmt.Println()
					renderPipelineConfigHistory(os.Stdout, history)
				}
				if includeSecrets {
					fmt.Println()
					renderPipelineSecrets(os.Stdout, secrets, showIDs)
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
	fs.BoolVar(&onlyConfig, "only-config", false, "Only show the pipeline configuration")
	fs.BoolVar(&includeEndpoints, "include-endpoints", false, "Include endpoints in output (only available with table format)")
	fs.BoolVar(&includeConfigHistory, "include-config-history", false, "Include config history in output (only available with table format)")
	fs.BoolVar(&includeSecrets, "include-secrets", false, "Include secrets in output (only available with table format)")
	fs.Uint64Var(&lastEndpoints, "last-endpoints", 0, "Last `N` pipeline endpoints if included. 0 means no limit")
	fs.Uint64Var(&lastConfigHistory, "last-config-history", 0, "Last `N` pipeline config history if included. 0 means no limit")
	fs.Uint64Var(&lastSecrets, "last-secrets", 0, "Last `N` pipeline secrets if included. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")

	fs.BoolVar(&showIDs, "show-ids", false, "Include IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func (config *config) fetchAllPipelines() ([]cloud.Pipeline, error) {
	aa, err := config.cloud.Aggregators(config.ctx, config.projectID, cloud.AggregatorsParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch aggregators: %w", err)
	}

	if len(aa) == 0 {
		return nil, nil
	}

	var pipelines []cloud.Pipeline
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.ctx)
	for _, a := range aa {
		a := a
		g.Go(func() error {
			got, err := config.cloud.Pipelines(gctx, a.ID, cloud.PipelinesParams{})
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

	var uniquePipelines []cloud.Pipeline
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
func pipelinesKeys(aa []cloud.Pipeline) []string {
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
	pp, err := config.cloud.ProjectPipelines(config.ctx, config.projectID, cloud.PipelinesParams{
		Name: &pipelineKey,
		Last: ptrUint64(2),
	})
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
