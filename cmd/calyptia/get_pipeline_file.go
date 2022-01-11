package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdGetPipelineFiles(config *config) *cobra.Command {
	var pipelineKey string
	var last uint64
	var format string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_files",
		Short: "Get pipeline files",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.cloud.PipelineFiles(config.ctx, pipelineID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline files: %w", err)
			}

			switch format {
			case "table":
				renderPipelineFiles(os.Stdout, ff, showIDs)
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(ff)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline files: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipeline files. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineFiles(w io.Writer, ff []cloud.PipelineFile, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "NAME\tENCRYPTED\tAGE")
	for _, f := range ff {
		if showIDs {
			fmt.Fprintf(tw, "%s\t", f.ID)
		}
		fmt.Fprintf(tw, "%s\t%v\t%s\n", f.Name, f.Encrypted, fmtAgo(f.CreatedAt))
	}
	tw.Flush()
}
