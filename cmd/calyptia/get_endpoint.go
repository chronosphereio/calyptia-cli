package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newCmdGetEndpoints(config *config) *cobra.Command {
	var format string
	var pipelineKey string
	var last uint64
	cmd := &cobra.Command{
		Use:   "endpoints",
		Short: "Display latest endpoints from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID := pipelineKey
			{
				pp, err := config.fetchAllPipelines()
				if err != nil {
					return err
				}

				pip, ok := findPipelineByName(pp, pipelineKey)
				if !ok && !validUUID(pipelineID) {
					return fmt.Errorf("could not find pipeline %q", pipelineKey)
				}

				if ok {
					pipelineID = pip.ID
				}
			}

			pp, err := config.cloud.PipelinePorts(config.ctx, pipelineID, last)
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline ports: %w", err)
			}

			switch format {
			case "table":
				tw := table.NewWriter()
				tw.AppendHeader(table.Row{"Protocol", "Frontend port", "Backend port", "Endpoint", "Age"})
				tw.Style().Options = table.OptionsNoBordersAndSeparators
				if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
					tw.SetAllowedRowLength(w)
				}

				for _, p := range pp {
					endpoint := p.Endpoint
					if endpoint == "" {
						endpoint = "Pending"
					}
					tw.AppendRow(table.Row{p.Protocol, p.FrontendPort, p.BackendPort, endpoint, fmtAgo(p.CreatedAt)})
				}
				fmt.Println(tw.Render())
			case "json":
				err := json.NewEncoder(os.Stdout).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline ports: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipeline endpoints. 0 means no limit")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
