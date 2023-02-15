package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/calyptia/utils"
)

func newCmdGetTraceSessions(config *utils.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var before string
	var showIDs bool
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:   "trace_sessions", // child of `get`
		Short: "List trace sessions",
		Long: "List all trace sessions from the given pipeline,\n" +
			"sorted by creation time in descending order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			var lastOpt *uint
			var beforeOpt *string

			if last > 0 {
				lastOpt = &last
			}
			if before != "" {
				beforeOpt = &before
			}

			ss, err := config.Cloud.TraceSessions(config.Ctx, pipelineID, types.TraceSessionsParams{
				Last:   lastOpt,
				Before: beforeOpt,
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, ss.Items)
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
	fs.UintVarP(&last, "last", "l", 0, "Last `N` trace sessions. 0 means no limit")
	fs.StringVar(&before, "before", "", "Only show trace sessions created before the given cursor")
	fs.BoolVar(&showIDs, "show-ids", false, "Show trace session IDs. Only applies when output format is table")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("pipeline")
	_ = cmd.RegisterFlagCompletionFunc("output-format", utils.CompleteOutputFormat)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.CompletePipelines)

	return cmd
}

func newCmdGetTraceSession(config *utils.Config) *cobra.Command {
	var pipelineKey string
	var showID bool
	var outputFormat, goTemplate string

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
				session, err = config.Cloud.TraceSession(config.Ctx, sessionID)
				if err != nil {
					return err
				}
			} else {
				if pipelineKey == "" {
					return errors.New("flag needs an argument: --pipeline")
				}

				pipelineID, err := config.LoadPipelineID(pipelineKey)
				if err != nil {
					return err
				}

				session, err = config.Cloud.ActiveTraceSession(config.Ctx, pipelineID)
				if err != nil {
					return err
				}
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, session)
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
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("output-format", utils.CompleteOutputFormat)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.CompletePipelines)

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
