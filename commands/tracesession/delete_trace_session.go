package tracesession

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/formatters"
)

func NewCmdDeleteTraceSession(cfg *config.Config) *cobra.Command {
	var confirmed bool
	var pipelineKey string

	cmd := &cobra.Command{
		Use:   "trace_session", // child of `delete`
		Short: "Terminate current active trace session from pipeline",
		Long: "Terminate the current active trace session from the given pipeline.\n" +
			"It does so by reducing its lifespan to now, effectively terminating it.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if !confirmed {
				cmd.Printf("Are you sure you want to terminate the current active trace session for pipeline %q? (y/N) ", pipelineKey)
				ok, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !ok {
					cmd.Println("Aborted")
					return nil
				}
			}

			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			terminated, err := cfg.Cloud.TerminateActiveTraceSession(ctx, pipelineID)
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), terminated)
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
				return tw.Flush()
			}
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("pipeline")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	return cmd
}
