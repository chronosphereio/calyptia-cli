package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/calyptia/api/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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
