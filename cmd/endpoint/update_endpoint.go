package endpoint

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/calyptia/cli/cmd/coreinstance"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdUpdateEndpoint(config *cfg.Config) *cobra.Command {
	var protocol string
	var ports string
	var serviceType string

	cmd := &cobra.Command{
		Use:   "endpoint ENDPOINT",
		Short: "Update pipeline endpoint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var fport, bport uint
			var fpport, bpport *uint

			portID := args[0]

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

					port, err = strconv.ParseUint(ports[colon+1:], 10, 16)
					if err != nil {
						return fmt.Errorf("unable to parse frontend port number: %w", err)
					}
					bport = uint(port)
				}

				fpport = &fport
				bpport = &bport
			}

			var opts cloud.UpdatePipelinePort

			if bpport != nil {
				opts.BackendPort = bpport
			}

			if fpport != nil {
				opts.FrontendPort = fpport
			}

			if protocol != "" {
				opts.Protocol = &protocol
			}

			if serviceType != "" {
				if !coreinstance.ValidPipelinePortKind(serviceType) {
					return fmt.Errorf("invalid provided service type %s, options are: %s", serviceType, coreinstance.AllValidPortKinds())
				}
				k := cloud.PipelinePortKind(serviceType)
				opts.Kind = &k
			}

			err := config.Cloud.UpdatePipelinePort(config.Ctx, portID, opts)
			if err != nil {
				return fmt.Errorf("could not update your pipeline endpoint: %w", err)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&protocol, "protocol", "", "Endpoint protocol, tcp or tcps")
	fs.StringVar(&ports, "ports", "", "define frontend and backend port, either: [port] or [frotend]:[backend]")
	fs.StringVar(&serviceType, "service-type", "", fmt.Sprintf("Service type to use for the ports, options: %s", coreinstance.AllValidPortKinds()))

	// _ = cmd.RegisterFlagCompletionFunc("output-format", completeOutputFormat)
	// _ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
