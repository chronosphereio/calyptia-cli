package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/hako/durafmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdGetAgents(config *config) *cobra.Command {
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool
	var fleetKey, environment string

	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Display latest agents from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			var params cloud.AgentsParams

			params.Last = &last
			if environmentID != "" {
				params.EnvironmentID = &environmentID
			}

			fs := cmd.Flags()
			if fs.Changed("fleet") {
				fleedID, err := config.loadFleetID(fleetKey)
				if err != nil {
					return err
				}

				params.FleetID = &fleedID
			}

			aa, err := config.cloud.Agents(config.ctx, config.projectID, params)
			if err != nil {
				return fmt.Errorf("could not fetch your agents: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, aa.Items)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tTYPE\tENVIRONMENT\tFLEET-ID\tVERSION\tSTATUS\tAGE")
				for _, a := range aa.Items {
					status := agentStatus(a.LastMetricsAddedAt, time.Minute*-5)
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", a.Name, a.Type, a.EnvironmentName, zeroOfPtr(a.FleetID), a.Version, status, fmtTime(a.CreatedAt))
				}
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(aa.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(aa.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include agent IDs in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVar(&fleetKey, "fleet", "", "Filter agents from the following fleet only")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("fleet", config.completeFleets)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

func newCmdGetAgent(config *config) *cobra.Command {
	var outputFormat, goTemplate string
	var showIDs bool
	var onlyConfig bool
	var environment string

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Display a specific agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			agentKey := args[0]
			agentID, err := config.loadAgentID(agentKey, environmentID)
			if err != nil {
				return err
			}

			agent, err := config.cloud.Agent(config.ctx, agentID)
			if err != nil {
				return fmt.Errorf("could not fetch your agent: %w", err)
			}

			if onlyConfig {
				fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(agent.RawConfig))
				return nil
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, agent)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tTYPE\tENVIRONMENT\tFLEET-ID\tVERSION\tSTATUS\tAGE")
				status := agentStatus(agent.LastMetricsAddedAt, time.Minute*-5)
				if showIDs {
					fmt.Fprintf(tw, "%s\t", agent.ID)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", agent.Name, agent.Type, agent.EnvironmentName, zeroOfPtr(agent.FleetID), agent.Version, status, fmtTime(agent.CreatedAt))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(agent)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(agent)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&onlyConfig, "only-config", false, "Only show the agent configuration")
	fs.BoolVar(&showIDs, "show-ids", false, "Include agent IDs in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

func (config *config) completeAgents(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.Agents(config.ctx, config.projectID, cloud.AgentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return agentsKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

// agentsKeys returns unique agent names first and then IDs.
func agentsKeys(aa []cloud.Agent) []string {
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
		count, ok := namesCount[a.Name]
		if !ok {
			continue
		}

		if count == 1 {
			out = append(out, a.Name)
			continue
		}

		out = append(out, a.ID)
	}

	return out
}

func (config *config) loadAgentID(agentKey string, environmentID string) (string, error) {
	var err error

	var params cloud.AgentsParams

	params.Last = ptr(uint(2))
	params.Name = &agentKey

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := config.cloud.Agents(config.ctx, config.projectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !validUUID(agentKey) {
		if len(aa.Items) != 0 {
			return "", fmt.Errorf("ambiguous agent name %q, use ID instead", agentKey)
		}
		return "", fmt.Errorf("could not find agent %q", agentKey)
	}

	if len(aa.Items) == 1 {
		return aa.Items[0].ID, nil
	}

	return agentKey, nil
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
