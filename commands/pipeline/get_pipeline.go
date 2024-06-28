package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetPipelines(cfg *config.Config) *cobra.Command {
	var coreInstanceKey string
	var last uint
	var outputFormat, goTemplate, configFormat string
	var showIDs bool
	var environment string
	var renderWithConfigSections bool

	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Display latest pipelines from a core-instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}
			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			if configFormat != "" {
				if !isValidConfigFormat(configFormat) {
					return fmt.Errorf("not a valid config format: %s", configFormat)
				}
			}
			pp, err := cfg.Cloud.Pipelines(ctx, cloudtypes.PipelinesParams{
				Last:                     &last,
				RenderWithConfigSections: renderWithConfigSections,
				CoreInstanceID:           &coreInstanceID,
				ConfigFormat:             (*cloudtypes.ConfigFormat)(&configFormat),
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipelines: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, pp.Items)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprintf(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tSTRATEGY\tAGE")
				for _, p := range pp.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", p.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", p.Name, p.ReplicasCount, p.Status.Status, string(p.DeploymentStrategy), formatters.FmtTime(p.CreatedAt))
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
	fs.StringVar(&configFormat, "config-format", string(cloudtypes.ConfigFormatYAML), "Format to get the configuration file from the API (yaml/json/ini).")

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance") // TODO: use default core instance ID from config cmd.

	return cmd
}

func NewCmdGetPipeline(cfg *config.Config) *cobra.Command {
	var onlyConfig bool
	var lastEndpoints, lastConfigHistory, lastSecrets uint
	var includeEndpoints, includeConfigHistory, includeSecrets bool
	var showIDs bool
	var renderWithConfigSections bool
	var outputFormat, goTemplate, configFormat string

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompletePipelines,
		Short:             "Display a pipelines by ID or name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineKey := args[0]
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			var pip cloudtypes.Pipeline
			var ports []cloudtypes.PipelinePort
			var configHistory []cloudtypes.PipelineConfig
			var secrets []cloudtypes.PipelineSecret

			if configFormat != "" {
				if !isValidConfigFormat(configFormat) {
					return fmt.Errorf("not a valid config format: %s", configFormat)
				}
			}

			if outputFormat == "table" && (includeEndpoints || includeConfigHistory || includeSecrets) && !onlyConfig {
				g, gctx := errgroup.WithContext(ctx)
				g.Go(func() error {
					var err error
					pip, err = cfg.Cloud.Pipeline(ctx, pipelineID, cloudtypes.PipelineParams{
						RenderWithConfigSections: renderWithConfigSections,
						ConfigFormat:             (*cloudtypes.ConfigFormat)(&configFormat),
					})
					if err != nil {
						return fmt.Errorf("could not fetch your pipeline: %w", err)
					}
					return nil
				})
				if includeEndpoints {
					g.Go(func() error {
						pp, err := cfg.Cloud.PipelinePorts(gctx, pipelineID, cloudtypes.PipelinePortsParams{
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
						cc, err := cfg.Cloud.PipelineConfigHistory(gctx, pipelineID, cloudtypes.PipelineConfigHistoryParams{
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
						ss, err := cfg.Cloud.PipelineSecrets(gctx, pipelineID, cloudtypes.PipelineSecretsParams{
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
				pip, err = cfg.Cloud.Pipeline(ctx, pipelineID, cloudtypes.PipelineParams{
					RenderWithConfigSections: renderWithConfigSections,
					ConfigFormat:             (*cloudtypes.ConfigFormat)(&configFormat),
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
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, pip)
			}

			switch outputFormat {
			case "table":
				{
					tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
					if showIDs {
						fmt.Fprint(tw, "ID\t")
					}
					fmt.Fprintln(tw, "NAME\tREPLICAS\tSTATUS\tSTRATEGY\tAGE")
					if showIDs {
						fmt.Fprintf(tw, "%s\t", pip.ID)
					}
					fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", pip.Name, pip.ReplicasCount, pip.Status.Status, string(pip.DeploymentStrategy), formatters.FmtTime(pip.CreatedAt))
					tw.Flush()
				}
				if includeEndpoints {
					fmt.Fprintln(cmd.OutOrStdout(), "\n## Endpoints")
					formatters.RenderEndpointsTable(cmd.OutOrStdout(), ports, showIDs)
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
	fs.StringVar(&configFormat, "config-format", string(cloudtypes.ConfigFormatYAML), "Format to get the configuration file from the API (yaml/json/ini).")
	fs.BoolVar(&showIDs, "show-ids", false, "Include IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

var allValidConfigFormats = []cloudtypes.ConfigFormat{
	cloudtypes.ConfigFormatYAML,
	cloudtypes.ConfigFormatJSON,
	cloudtypes.ConfigFormatINI,
}

func isValidConfigFormat(s string) bool {
	for _, val := range allValidConfigFormats {
		if val == cloudtypes.ConfigFormat(s) {
			return true
		}
	}
	return false
}
