package tracerecord

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetTraceRecords(config *cfg.Config) *cobra.Command {
	var sessionID string
	var last uint
	var before string
	var showIDs bool
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}
	cmd := &cobra.Command{
		Use:   "trace_records", // child of `create`
		Short: "List trace records",
		Long: "List all records from the given trace session,\n" +
			"sorted by creation time in descending order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var lastOpt *uint
			var beforeOpt *string

			if last > 0 {
				lastOpt = &last
			}
			if before != "" {
				beforeOpt = &before
			}

			ss, err := config.Cloud.TraceRecords(config.Ctx, sessionID, types.TraceRecordsParams{
				Last:   lastOpt,
				Before: beforeOpt,
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, ss.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ss)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ss)
			default:
				return renderTraceRecordsTable(cmd.OutOrStdout(), ss, sessionID, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&sessionID, "session", "", "Parent trace session ID from which to list the records")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` trace records. 0 means no limit")
	fs.StringVar(&before, "before", "", "Only show trace records created before the given cursor")
	fs.BoolVar(&showIDs, "show-ids", false, "Show trace records IDs. Only applies when output format is table")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("session")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("session", completer.CompleteTraceSessions)

	return cmd
}

func renderTraceRecordsTable(w io.Writer, rr types.TraceRecords, sessionID string, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	// TODO: show actual records in a nice human readable way.
	// Maybe logfmt.
	fmt.Fprintln(tw, "TYPE\tTRACE-ID\tSTART\tEND\tPLUGIN-ID\tPLUGIN-ALIAS\tRETURN-CODE\tAGE")
	for _, rec := range rr.Items {
		if showIDs {
			_, err := fmt.Fprintf(tw, "%s\t", rec.ID)
			if err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			fmtTraceRecordKind(rec.Kind), rec.TraceID,
			utils.FmtTime(rec.StartTime), utils.FmtTime(rec.EndTime),
			rec.PluginInstance, rec.PluginAlias, rec.ReturnCode,
			utils.FmtTime(rec.CreatedAt),
		)
		if err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if rr.EndCursor != nil {
		_, err := fmt.Fprintf(w, "\n\n# Previous page:\n\tcalyptia get trace_records --session %s --before %s\n", sessionID, *rr.EndCursor)
		if err != nil {
			return err
		}
	}

	return nil
}

func fmtTraceRecordKind(kind types.TraceRecordKind) string {
	switch kind {
	case types.TraceRecordKindInput:
		return "input"
	case types.TraceRecordKindFilter:
		return "filter"
	case types.TraceRecordKindPreOutput:
		return "pre-output"
	case types.TraceRecordKindOutput:
		return "output"
	}
	return "unknown"
}
