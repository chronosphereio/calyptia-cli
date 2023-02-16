package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdGetPipelines(config *config) *cobra.Command {
	var coreInstanceKey string
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool
	var environment string
	var renderWithConfigSections bool

	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Display latest pipelines from a core-instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			coreInstanceID, err := config.loadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			pp, err := config.cloud.Pipelines(config.ctx, coreInstanceID, cloud.PipelinesParams{
				Last:                     &last,
				RenderWithConfigSections: renderWithConfigSections,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, pp.Items)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tAGE")
				for _, p := range pp.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", p.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", p.Name, p.ReplicasCount, p.Status.Status, fmtTime(p.CreatedAt))
				}
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(pp.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipelines. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include pipeline IDs in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.BoolVar(&renderWithConfigSections, "render-with-config-sections", false, "Render the pipeline config with the attached config sections; if any")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", config.completeCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default core instance ID from config cmd.

	return cmd
}

func newCmdGetPipeline(config *config) *cobra.Command {
	var onlyConfig bool
	var lastEndpoints, lastConfigHistory, lastSecrets uint
	var includeEndpoints, includeConfigHistory, includeSecrets bool
	var showIDs bool
	var renderWithConfigSections bool
	var outputFormat, goTemplate string

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
			var configHistory []cloud.PipelineConfig
			var secrets []cloud.PipelineSecret
			if outputFormat == "table" && (includeEndpoints || includeConfigHistory || includeSecrets) && !onlyConfig {
				g, gctx := errgroup.WithContext(config.ctx)
				g.Go(func() error {
					var err error
					pip, err = config.cloud.Pipeline(config.ctx, pipelineID, cloud.PipelineParams{
						RenderWithConfigSections: renderWithConfigSections,
					})
					if err != nil {
						return fmt.Errorf("could not fetch your pipeline: %w", err)
					}
					return nil
				})
				if includeEndpoints {
					g.Go(func() error {
						pp, err := config.cloud.PipelinePorts(gctx, pipelineID, cloud.PipelinePortsParams{
							Last: &lastEndpoints,
						})
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline endpoints: %w", err)
						}

						ports = pp.Items
						return nil
					})
				}
				if includeConfigHistory {
					g.Go(func() error {
						cc, err := config.cloud.PipelineConfigHistory(gctx, pipelineID, cloud.PipelineConfigHistoryParams{
							Last: &lastConfigHistory,
						})
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline config history: %w", err)
						}

						configHistory = cc.Items
						return nil
					})
				}
				if includeSecrets {
					g.Go(func() error {
						ss, err := config.cloud.PipelineSecrets(gctx, pipelineID, cloud.PipelineSecretsParams{
							Last: &lastSecrets,
						})
						if err != nil {
							return fmt.Errorf("could not fetch your pipeline secrets: %w", err)
						}
						secrets = ss.Items
						return nil
					})
				}

				if err := g.Wait(); err != nil {
					return err
				}
			} else {
				var err error
				pip, err = config.cloud.Pipeline(config.ctx, pipelineID, cloud.PipelineParams{
					RenderWithConfigSections: renderWithConfigSections,
				})
				if err != nil {
					return fmt.Errorf("could not fetch your pipeline: %w", err)
				}
			}

			if onlyConfig {
				fmt.Println(strings.TrimSpace(pip.Config.RawConfig))
				return nil
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, pip)
			}

			switch outputFormat {
			case "table":
				{
					tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
					if showIDs {
						fmt.Fprint(tw, "ID\t")
					}
					fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tAGE")
					if showIDs {
						fmt.Fprintf(tw, "%s\t", pip.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n", pip.Name, pip.ReplicasCount, pip.Status.Status, fmtTime(pip.CreatedAt))
					tw.Flush()
				}
				if includeEndpoints {
					fmt.Fprintln(cmd.OutOrStdout(), "\n## Endpoints")
					renderEndpointsTable(cmd.OutOrStdout(), ports, showIDs)
				}
				if includeConfigHistory {
					fmt.Fprintln(cmd.OutOrStdout(), "\n## Configuration History")
					renderPipelineConfigHistory(cmd.OutOrStdout(), configHistory)
				}
				if includeSecrets {
					fmt.Fprintln(cmd.OutOrStdout(), "\n## Secrets")
					renderPipelineSecrets(cmd.OutOrStdout(), secrets, showIDs)
				}
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(pip)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(pip)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&onlyConfig, "only-config", false, "Only show the pipeline configuration")
	fs.BoolVar(&includeEndpoints, "include-endpoints", false, "Include endpoints in output (only available with table format)")
	fs.BoolVar(&includeConfigHistory, "include-config-history", false, "Include config history in output (only available with table format)")
	fs.BoolVar(&includeSecrets, "include-secrets", false, "Include secrets in output (only available with table format)")
	fs.UintVar(&lastEndpoints, "last-endpoints", 0, "Last `N` pipeline endpoints if included. 0 means no limit")
	fs.UintVar(&lastConfigHistory, "last-config-history", 0, "Last `N` pipeline config history if included. 0 means no limit")
	fs.UintVar(&lastSecrets, "last-secrets", 0, "Last `N` pipeline secrets if included. 0 means no limit")
	fs.BoolVar(&renderWithConfigSections, "render-with-config-sections", false, "Render the pipeline config with the attached config sections; if any")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	fs.BoolVar(&showIDs, "show-ids", false, "Include IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

func (config *config) fetchAllPipelines() ([]cloud.Pipeline, error) {
	aa, err := config.cloud.CoreInstances(config.ctx, config.projectID, cloud.CoreInstancesParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch core-instances: %w", err)
	}

	if len(aa.Items) == 0 {
		return nil, nil
	}

	var pipelines []cloud.Pipeline
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.ctx)
	for _, a := range aa.Items {
		a := a
		g.Go(func() error {
			got, err := config.cloud.Pipelines(gctx, a.ID, cloud.PipelinesParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			pipelines = append(pipelines, got.Items...)
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
		cobra.CompError(err.Error())
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
		Last: ptr(uint(2)),
	})
	if err != nil {
		return "", err
	}

	if len(pp.Items) != 1 && !validUUID(pipelineKey) {
		if len(pp.Items) != 0 {
			return "", fmt.Errorf("ambiguous pipeline name %q, use ID instead", pipelineKey)
		}

		return "", fmt.Errorf("could not find pipeline %q", pipelineKey)
	}

	if len(pp.Items) == 1 {
		return pp.Items[0].ID, nil
	}

	return pipelineKey, nil
}
