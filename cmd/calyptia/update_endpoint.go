package main

import (
	"fmt"

	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
)

func newCmdUpdateEndpoint(config *config) *cobra.Command {
	var protocol string
	var port uint
	var portID string

	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "Update pipeline endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := cloud.UpdatePipelinePort{
				BackendPort:  &port,
				FrontendPort: &port,
				Protocol:     &protocol,
			}
			err := config.cloud.UpdatePipelinePort(config.ctx, portID, opts)
			if err != nil {
				return fmt.Errorf("could not update your pipeline endpoint: %w", err)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&protocol, "protocol", "", "Endpoint protocol, tcp or tcps")
	fs.UintVar(&port, "port", 0, "port")
	fs.StringVar(&portID, "id", "", "Endpoint port ID")

	// _ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	// _ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
