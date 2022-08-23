package main

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdGetPipelineFiles(config *config) *cobra.Command {
	var pipelineKey string
	var last uint
	var format string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "pipeline_files",
		Short: "Get pipeline files",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.cloud.PipelineFiles(config.ctx, pipelineID, cloud.PipelineFilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your pipeline files: %w", err)
			}

			switch format {
			case "table":
				renderPipelineFiles(cmd.OutOrStdout(), ff.Items, showIDs)
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline files: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` pipeline files. 0 means no limit")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")

	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}

func newCmdGetPipelineFile(config *config) *cobra.Command {
	var pipelineKey string
	var name string
	var format string
	var showIDs, onlyContents bool

	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Get a single file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.cloud.PipelineFiles(config.ctx, pipelineID, cloud.PipelineFilesParams{})
			if err != nil {
				return fmt.Errorf("could not find your pipeline file by name: %w", err)
			}

			var file cloud.PipelineFile
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
				fmt.Print(file.Contents)
				return nil
			}

			switch format {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tENCRYPTED\tAGE")
				if showIDs {
					fmt.Fprintf(tw, "%s\t", file.ID)
				}
				fmt.Fprintf(tw, "%s\t%v\t%s\n", file.Name, file.Encrypted, fmtAgo(file.CreatedAt))
				tw.Flush()
			case "json":
				err := json.NewEncoder(cmd.OutOrStdout()).Encode(file)
				if err != nil {
					return fmt.Errorf("could not json encode your pipeline file: %w", err)
				}
			default:
				return fmt.Errorf("unknown output format %q", format)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&name, "name", "", "File name")
	fs.StringVarP(&format, "output-format", "o", "table", "Output format. Allowed: table, json")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.BoolVar(&onlyContents, "only-contents", false, "Only print file contents")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)
	_ = cmd.RegisterFlagCompletionFunc("output-format", config.completeOutputFormat)

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func renderPipelineFiles(w io.Writer, ff []cloud.PipelineFile, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "NAME\tENCRYPTED\tAGE")
	for _, f := range ff {
		if showIDs {
			fmt.Fprintf(tw, "%s\t", f.ID)
		}
		fmt.Fprintf(tw, "%s\t%v\t%s\n", f.Name, f.Encrypted, fmtAgo(f.CreatedAt))
	}
	tw.Flush()
}
