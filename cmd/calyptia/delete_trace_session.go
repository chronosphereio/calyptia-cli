package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newCmdDeleteTraceSession(config *config) *cobra.Command {
	var confirmed bool
	var pipelineKey string
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "trace_session",
		Short: "Terminate current active trace session from pipeline",
		Long: "Terminate the current active trace session from the given pipeline.\n" +
			"It does so by reducing its lifespan to now, effectively terminating it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				cmd.Printf("Are you sure you want to terminate the current active trace session for pipeline %q? (y/N) ", pipelineKey)
				ok, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !ok {
					return nil
				}
			}

			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			terminated, err := config.cloud.TerminateActiveTraceSession(config.ctx, pipelineID)
			if err != nil {
				return err
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(terminated)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(terminated)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID")
				fmt.Fprintf(tw, "%s\n", terminated.ID)
				tw.Flush()

				return nil
			}
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&outputFormat, "output", "table", "Output format (table, json, yaml)")

	_ = cmd.MarkFlagRequired("pipeline")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	return cmd
}

func readConfirm(r io.Reader) (bool, error) {
	var answer string
	_, err := fmt.Fscanln(r, &answer)
	if err != nil && err.Error() == "unexpected newline" {
		err = nil
	}

	if err != nil {
		return false, fmt.Errorf("could not to read answer: %v", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}
