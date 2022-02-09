package main

import (
	"fmt"
	"strconv"
	"strings"

	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
)

func newCmdUpdateEndpoint(config *config) *cobra.Command {
	var protocol string
	var portID string
	var ports string

	cmd := &cobra.Command{
		Use:   "endpoint",
		Short: "Update pipeline endpoint",
		RunE: func(cmd *cobra.Command, args []string) error {
			var fport, bport uint
			var fpport, bpport *uint

			if ports != "" {
				colon := strings.Index(ports, ":")
				if colon == -1 {
					port, err := strconv.ParseUint(ports, 10, 16)
					if err != nil {
						return fmt.Errorf("unable to parse port number: %w", err)
					}
					bport = uint(port)
					fport = uint(port)
				} else {
					port, err := strconv.ParseUint(ports[0:colon], 10, 16)
					if err != nil {
						return fmt.Errorf("unable to parse frontend port number: %w", err)
					}
					fport = uint(port)

					port, err = strconv.ParseUint(ports[colon+1:len(ports)], 10, 16)
					if err != nil {
						return fmt.Errorf("unable to parse frontend port number: %w", err)
					}
					bport = uint(port)
				}

				fpport = &fport
				bpport = &bport
			}

			opts := cloud.UpdatePipelinePort{
				BackendPort:  bpport,
				FrontendPort: fpport,
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
	fs.StringVar(&ports, "ports", "", "define frontend and backend port, either: [port] or [frotend]:[backend]")
	fs.StringVar(&portID, "id", "", "Endpoint port ID")

	// _ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	// _ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
