package tracesession

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/formatters"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
)

func NewCmdDeleteTraceSession(config *cfg.Config) *cobra.Command {
	var confirmed bool
	var pipelineKey string
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "trace_session", // child of `delete`
		Short: "Terminate current active trace session from pipeline",
		Long: "Terminate the current active trace session from the given pipeline.\n" +
			"It does so by reducing its lifespan to now, effectively terminating it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				cmd.Printf("Are you sure you want to terminate the current active trace session for pipeline %q? (y/N) ", pipelineKey)
				ok, err := confirm.ReadConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !ok {
					cmd.Println("Aborted")
					return nil
				}
			}

			pipelineID, err := completer.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			terminated, err := config.Cloud.TerminateActiveTraceSession(config.Ctx, pipelineID)
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, terminated)
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

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("pipeline")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", completer.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
