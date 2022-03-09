package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdDeletePipelineFile(config *config) *cobra.Command {
	var confirmed bool
	var pipelineKey string
	var name string

	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Delete a single file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				fmt.Printf("Are you sure you want to delete %q? (y/N) ", pipelineKey)
				var answer string
				_, err := fmt.Scanln(&answer)
				if err != nil && err.Error() == "unexpected newline" {
					err = nil
				}

				if err != nil {
					return fmt.Errorf("could not to read answer: %v", err)
				}

				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					return nil
				}
			}

			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.cloud.PipelineFiles(config.ctx, pipelineID, cloud.PipelineFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return config.cloud.DeletePipelineFile(config.ctx, f.ID)
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&name, "name", "", "File name you want to delete")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)
	_ = cmd.MarkFlagRequired("name")

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
