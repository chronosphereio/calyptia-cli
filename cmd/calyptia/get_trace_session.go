package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
)

func newCmdGetTraceSessions(config *config) *cobra.Command {
	var pipelineKey string
	var last uint64
	var before string
	var showIDs bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "trace_sessions", // child of `create`
		Short: "List trace sessions",
		Long: "List all trace sessions from the given pipeline,\n" +
			"sorted by creation time in descending order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			var lastOpt *uint64
			var beforeOpt *string

			if last > 0 {
				lastOpt = &last
			}
			if before != "" {
				beforeOpt = &before
			}

			ss, err := config.cloud.TraceSessions(config.ctx, pipelineID, types.TraceSessionsParams{
				Last:   lastOpt,
				Before: beforeOpt,
			})
			if err != nil {
				return err
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ss)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ss)
			default:
				return renderTraceSessionsTable(cmd.OutOrStdout(), ss, pipelineID, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline (name or ID) from which to list the trace sessions")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` trace sessions. 0 means no limit")
	fs.StringVar(&before, "before", "", "Only show trace sessions created before the given cursor")
	fs.BoolVar(&showIDs, "show-ids", false, "Show trace session IDs. Only applies when output format is table")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml")

	_ = cmd.MarkFlagRequired("pipeline")
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	return cmd
}

func newCmdGetTraceSession(config *config) *cobra.Command {
	var pipelineKey string
	var showID bool
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "trace_session TRACE_SESSION", // child of `get`
		Short: "Get a single trace session",
		Long: "Get a single trace session either by passing its name or ID,\n" +
			"or getting the current active trace session from the given pipeline.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var session types.TraceSession
			if len(args) == 1 {
				sessionID := args[0]
				var err error
				session, err = config.cloud.TraceSession(config.ctx, sessionID)
				if err != nil {
					return err
				}
			} else {
				if pipelineKey == "" {
					return errors.New("flag needs an argument: --pipeline")
				}

				pipelineID, err := config.loadPipelineID(pipelineKey)
				if err != nil {
					return err
				}

				session, err = config.cloud.ActiveTraceSession(config.ctx, pipelineID)
				if err != nil {
					return err
				}
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(session)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(session)
			default:
				return renderTraceSessionTable(cmd.OutOrStdout(), session, showID)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline (name or ID) from which to fetch the current active trace session. Only required if TRACE_SESSION argument is not provided")
	fs.BoolVar(&showID, "show-id", false, "Show trace session ID. Only applies when output format is table")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	return cmd
}

func renderTraceSessionsTable(w io.Writer, ss types.TraceSessions, pipelineID string, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	fmt.Fprintln(tw, "PLUGINS\tLIFESPAN\tACTIVE\tAGE")
	for _, sess := range ss.Items {
		if showIDs {
			_, err := fmt.Fprintf(tw, "%s\t", sess.ID)
			if err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%v\t%v\t%s\n", strings.Join(sess.Plugins, ", "), fmtDuration(time.Duration(sess.Lifespan)), sess.Active(), fmtTime(sess.CreatedAt))
		if err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if ss.EndCursor != nil {
		_, err := fmt.Fprintf(w, "\n\n# Previous page:\n\tcalyptia get trace_sessions --pipeline %s --before %s\n", pipelineID, *ss.EndCursor)
		if err != nil {
			return err
		}
	}

	return nil
}

func renderTraceSessionTable(w io.Writer, sess types.TraceSession, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	fmt.Fprintln(tw, "PLUGINS\tLIFESPAN\tACTIVE\tAGE")
	if showIDs {
		_, err := fmt.Fprintf(tw, "%s\t", sess.ID)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(tw, "%s\t%v\t%v\t%s\n", strings.Join(sess.Plugins, ", "), fmtDuration(time.Duration(sess.Lifespan)), sess.Active(), fmtTime(sess.CreatedAt))
	if err != nil {
		return err
	}

	return tw.Flush()
}

func (config *config) completeTraceSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ss, err := config.fetchAllTraceSessions()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if ss == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	out := make([]string, len(ss))
	for i, p := range ss {
		out[i] = p.ID
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (config *config) fetchAllTraceSessions() ([]types.TraceSession, error) {
	pp, err := config.fetchAllPipelines()
	if err != nil {
		return nil, err
	}

	if len(pp) == 0 {
		return nil, nil
	}

	var ss []types.TraceSession
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.ctx)
	for _, pip := range pp {
		a := pip
		g.Go(func() error {
			got, err := config.cloud.TraceSessions(gctx, a.ID, types.TraceSessionsParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			ss = append(ss, got.Items...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return ss, nil
}
