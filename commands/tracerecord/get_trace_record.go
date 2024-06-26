package tracerecord

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetTraceRecords(cfg *config.Config) *cobra.Command {
	var sessionID string
	var last uint
	var before string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "trace_records", // child of `create`
		Short: "List trace records",
		Long: "List all records from the given trace session,\n" +
			"sorted by creation time in descending order.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var lastOpt *uint
			var beforeOpt *string

			if last > 0 {
				lastOpt = &last
			}
			if before != "" {
				beforeOpt = &before
			}

			ss, err := cfg.Cloud.TraceRecords(ctx, sessionID, cloudtypes.TraceRecordsParams{
				Last:   lastOpt,
				Before: beforeOpt,
			})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ss)
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
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("session")
	_ = cmd.RegisterFlagCompletionFunc("session", cfg.Completer.CompleteTraceSessions)

	return cmd
}

func renderTraceRecordsTable(w io.Writer, rr cloudtypes.TraceRecords, sessionID string, showIDs bool) error {
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
			formatters.FmtTime(rec.StartTime), formatters.FmtTime(rec.EndTime),
			rec.PluginInstance, rec.PluginAlias, rec.ReturnCode,
			formatters.FmtTime(rec.CreatedAt),
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

func fmtTraceRecordKind(kind cloudtypes.TraceRecordKind) string {
	switch kind {
	case cloudtypes.TraceRecordKindInput:
		return "input"
	case cloudtypes.TraceRecordKindFilter:
		return "filter"
	case cloudtypes.TraceRecordKindPreOutput:
		return "pre-output"
	case cloudtypes.TraceRecordKindOutput:
		return "output"
	}
	return "unknown"
}
