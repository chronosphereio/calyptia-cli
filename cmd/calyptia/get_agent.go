package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	cloud "github.com/calyptia/api/types"
	"github.com/hako/durafmt"
	"github.com/spf13/cobra"
)

func newCmdGetAgents(config *config) *cobra.Command {
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Display latest agents from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			aa, err := config.cloud.Agents(config.ctx, config.projectID, cloud.AgentsParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your agents: %w", err)
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tTYPE\tVERSION\tSTATUS\tAGE")
				for _, a := range aa {
					status := agentStatus(a.LastMetricsAddedAt, time.Minute*-5)
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", a.Name, a.Type, a.Version, status, fmtAgo(a.CreatedAt))
				}
				tw.Flush()
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(aa)
				if err != nil {
					return fmt.Errorf("could not json encode your agents: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include agent IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func newCmdGetAgent(config *config) *cobra.Command {
	var format string
	var showIDs bool
	var onlyConfig bool

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Display a specific agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentKey := args[0]
			agentID, err := config.loadAgentID(agentKey)
			if err != nil {
				return err
			}

			agent, err := config.cloud.Agent(config.ctx, agentID)
			if err != nil {
				return fmt.Errorf("could not fetch your agent: %w", err)
			}

			if onlyConfig {
				fmt.Println(strings.TrimSpace(agent.RawConfig))
				return nil
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tTYPE\tVERSION\tSTATUS\tAGE")
				status := agentStatus(agent.LastMetricsAddedAt, time.Minute*-5)
				if showIDs {
					fmt.Fprintf(tw, "%s\t", agent.ID)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", agent.Name, agent.Type, agent.Version, status, fmtAgo(agent.CreatedAt))
				tw.Flush()
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(agent)
				if err != nil {
					return fmt.Errorf("could not json encode your agent: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&onlyConfig, "only-config", false, "Only show the agent configuration")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include agent IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func (config *config) completeAgents(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var aa []cloud.Agent
	var err error
	aa, err = config.cloud.Agents(config.ctx, config.projectID, cloud.AgentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return agentsKeys(aa), cobra.ShellCompDirectiveNoFileComp
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

func (config *config) loadAgentID(agentKey string) (string, error) {
	var err error
	aa, err := config.cloud.Agents(config.ctx, config.projectID, cloud.AgentsParams{
		Name: &agentKey,
		Last: ptrUint64(2),
	})
	if err != nil {
		return "", err
	}

	if len(aa) != 1 && !validUUID(agentKey) {
		if len(aa) != 0 {
			return "", fmt.Errorf("ambiguous agent name %q, use ID instead", agentKey)
		}
		return "", fmt.Errorf("could not find agent %q", agentKey)
	}

	if len(aa) == 1 {
		return aa[0].ID, nil
	}

	return agentKey, nil
}

func agentStatus(lastMetricsAddedAt time.Time, start time.Duration) string {
	var status string
	if lastMetricsAddedAt.IsZero() {
		status = "inactive"
	} else if lastMetricsAddedAt.Before(time.Now().Add(start)) {
		status = fmt.Sprintf("inactive for %s", durafmt.ParseShort(time.Since(lastMetricsAddedAt)).LimitFirstN(1))
	} else {
		status = "active"
	}
	return status
}
