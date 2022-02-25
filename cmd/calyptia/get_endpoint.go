package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetEndpoints(config *config) *cobra.Command {
	var pipelineKey string
	var last uint64
	var format string
	var showIDs bool
	cmd := &cobra.Command{
		Use:   "endpoints",
		Short: "Display latest endpoints from a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			pp, err := config.cloud.PipelinePorts(config.ctx, pipelineID, cloud.PipelinePortsParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline endpoints: %w", err)
			}

			switch format {
			case "table":
				renderEndpointsTable(cmd.OutOrStdout(), pp, showIDs)
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(pp)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline endpoints: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` pipeline endpoints. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include endpoint IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderEndpointsTable(w io.Writer, pp []cloud.PipelinePort, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "PROTOCOL\tFRONTEND-PORT\tBACKEND-PORT\tENDPOINT\tAGE")
	for _, p := range pp {
		endpoint := p.Endpoint
		if endpoint == "" {
			endpoint = "Pending"
		}
		if showIDs {
			fmt.Fprintf(tw, "%s\t", p.ID)
		}
		fmt.Fprintf(tw, "%s\t%d\t%d\t%s\t%s\n", p.Protocol, p.FrontendPort, p.BackendPort, endpoint, fmtAgo(p.CreatedAt))
	}
	tw.Flush()
}
