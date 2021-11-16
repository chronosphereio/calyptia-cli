package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/campoy/unique"
	"github.com/hako/durafmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func newCmdGetAgents(config *config) *cobra.Command {
	var format string
	var projectID string
	var last uint64
	cmd := &cobra.Command{
		Use:   "agents",
		Short: "Display latest agents from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			aa, err := config.cloud.Agents(config.ctx, projectID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your agents: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"ID", "Name", "Type", "Version", "Status", "Created at"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, a := range aa {
					tw.AppendRow(table.Row{a.ID, a.Name, a.Type, a.Version, agentStatus(a.LastMetricsAddedAt, time.Minute*-5), a.CreatedAt})
				}
				fmt.Println(tw.Render())
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
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&projectID, "project-id", "", "Parent project ID")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("project-id", config.completeProjectIDs)

	_ = cmd.MarkFlagRequired("project-id") // TODO: use default project ID from config cmd.

	return cmd
}

func (config *config) completeAgentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.cloud.Projects(config.ctx, 0)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(pp) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var out []string
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, p := range pp {
		p := p
		g.Go(func() error {
			aa, err := config.cloud.Agents(gctx, p.ID, 0)
			if err != nil {
				return err
			}

			mu.Lock()
			for _, a := range aa {
				out = append(out, a.ID)
			}
			mu.Unlock()

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	unique.Slice(&out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return out, cobra.ShellCompDirectiveNoFileComp
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
