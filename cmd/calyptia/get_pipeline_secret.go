package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetPipelineSecrets(config *config) *cobra.Command {
	var pipelineKey string
	var last uint64
	var format string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_secrets",
		Short: "Get pipeline secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ss, err := config.cloud.PipelineSecrets(config.ctx, pipelineID, cloud.PipelineSecretsParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline secrets: %w", err)
			}

			switch format {
			case "table":
				renderPipelineSecrets(cmd.OutOrStdout(), ss.Items, showIDs)
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline secrets: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipeline secrets. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineSecrets(w io.Writer, ss []cloud.PipelineSecret, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "KEY\tAGE")
	for _, s := range ss {
		if showIDs {
			fmt.Fprintf(tw, "%s\t", s.ID)
		}
		fmt.Fprintf(tw, "%s\t%s\n", s.Key, fmtTime(s.CreatedAt))
	}
	tw.Flush()
}
