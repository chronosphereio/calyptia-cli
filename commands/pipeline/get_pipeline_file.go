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

func NewCmdGetPipelineFiles(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_files",
		Short: "Get pipeline files",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.PipelineFiles(ctx, pipelineID, cloudtypes.PipelineFilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline files: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ff.Items)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			default:
				return renderPipelineFiles(cmd.OutOrStdout(), ff.Items, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline files. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func NewCmdGetPipelineFile(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var name string
	var showIDs, onlyContents bool

	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Get a single file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.PipelineFiles(ctx, pipelineID, cloudtypes.PipelineFilesParams{})
			if err != nil {
				return fmt.Errorf("could not find your pipeline file by name: %w", err)
			}

			var file cloudtypes.PipelineFile
			var found bool
			for _, f := range ff.Items {
				if f.Name == name {
					file = f
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("could not find pipeline file %q", name)
			}

			if onlyContents {
				if !file.Encrypted {
					cmd.Print(string(file.Contents))
				} else {
					cmd.Print(file.Contents)
				}
				return nil
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), file)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(file)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(file)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tENCRYPTED\tAGE")
				if showIDs {
					fmt.Fprintf(tw, "%s\t", file.ID)
				}
				fmt.Fprintf(tw, "%s\t%v\t%s\n", file.Name, file.Encrypted, formatters.FmtTime(file.CreatedAt))
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&name, "name", "", "File name")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.BoolVar(&onlyContents, "only-contents", false, "Only print file contents")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func renderPipelineFiles(w io.Writer, ff []cloudtypes.PipelineFile, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(tw, "NAME\tENCRYPTED\tAGE"); err != nil {
		return err
	}

	for _, f := range ff {
		if showIDs {
			if _, err := fmt.Fprintf(tw, "%s\t", f.ID); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%v\t%s\n", f.Name, f.Encrypted, formatters.FmtTime(f.CreatedAt))
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}
