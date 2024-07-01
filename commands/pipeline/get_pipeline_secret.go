package pipeline

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetPipelineSecrets(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_secrets",
		Short: "Get pipeline secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			ss, err := cfg.Cloud.PipelineSecrets(ctx, pipelineID, cloudtypes.PipelineSecretsParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline secrets: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ss.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ss.Items)
			default:
				return renderPipelineSecrets(cmd.OutOrStdout(), ss.Items, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline secrets. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func renderPipelineSecrets(w io.Writer, ss []cloudtypes.PipelineSecret, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(tw, "KEY\tAGE"); err != nil {
		return err
	}
	for _, s := range ss {
		if showIDs {
			if _, err := fmt.Fprintf(tw, "%s\t", s.ID); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%s\n", s.Key, formatters.FmtTime(s.CreatedAt))
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}
