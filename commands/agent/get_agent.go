package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hako/durafmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
	"github.com/calyptia/cli/pointer"
)

func NewCmdGetAgents(cfg *config.Config) *cobra.Command {
	var last uint
	var showIDs bool
	var fleetKey, environment string

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Display latest agents from a project",
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
			var params cloudtypes.AgentsParams

			params.Last = &last
			if environmentID != "" {
				params.EnvironmentID = &environmentID
			}

			fs := cmd.Flags()
			if fs.Changed("fleet") {
				fleedID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
				if err != nil {
					return err
				}

				params.FleetID = &fleedID
			}

			out, err := cfg.Cloud.Agents(ctx, cfg.ProjectID, params)
			if err != nil {
				return fmt.Errorf("could not fetch your agents: %w", err)
			}

			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), out)
			}

			switch outputFormat {
			case formatters.OutputFormatJSON:
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out.Items)
			case formatters.OutputFormatYAML:
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out.Items)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tTYPE\tENVIRONMENT\tFLEET-ID\tVERSION\tSTATUS\tAGE")
				for _, a := range out.Items {
					status := agentStatus(a.LastMetricsAddedAt, time.Minute*-5)
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", a.Name, a.Type, a.EnvironmentName, pointer.OrZero(a.FleetID), a.Version, status, formatters.FmtTime(a.CreatedAt))
				}
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include agent IDs in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVar(&fleetKey, "fleet", "", "Filter agents from the following fleet only")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("fleet", cfg.Completer.CompleteFleets)

	return cmd
}

func NewCmdGetAgent(cfg *config.Config) *cobra.Command {
	var showIDs bool
	var onlyConfig bool

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Display a specific agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			agentKey := args[0]
			agentID, err := cfg.Completer.LoadAgentID(ctx, agentKey)
			if err != nil {
				return err
			}

			agent, err := cfg.Cloud.Agent(ctx, agentID)
			if err != nil {
				return fmt.Errorf("could not fetch your agent: %w", err)
			}

			if onlyConfig {
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(agent.RawConfig))
				return nil
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), agent)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(agent)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(agent)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tTYPE\tENVIRONMENT\tFLEET-ID\tVERSION\tSTATUS\tAGE")
				status := agentStatus(agent.LastMetricsAddedAt, time.Minute*-5)
				if showIDs {
					fmt.Fprintf(tw, "%s\t", agent.ID)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", agent.Name, agent.Type, agent.EnvironmentName, pointer.OrZero(agent.FleetID), agent.Version, status, formatters.FmtTime(agent.CreatedAt))
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&onlyConfig, "only-config", false, "Only show the agent configuration")
	fs.BoolVar(&showIDs, "show-ids", false, "Include agent IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)

	return cmd
}

func agentStatus(lastMetricsAddedAt *time.Time, start time.Duration) string {
	var status string
	if lastMetricsAddedAt == nil || lastMetricsAddedAt.IsZero() {
		status = "inactive"
	} else if lastMetricsAddedAt.Before(time.Now().Add(start)) {
		status = fmt.Sprintf("inactive for %s", durafmt.ParseShort(time.Since(*lastMetricsAddedAt)).LimitFirstN(1))
	} else {
		status = "active"
	}
	return status
}
