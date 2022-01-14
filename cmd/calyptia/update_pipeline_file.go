package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdUpdatePipelineFile(config *config) *cobra.Command {
	var pipelineKey string
	var file string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Update a file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			contents, err := readFile(file)
			if err != nil {
				return err
			}

			name := filepath.Base(file)
			name = strings.TrimSuffix(name, filepath.Ext(name))

			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.cloud.PipelineFiles(config.ctx, pipelineID, 0)
			if err != nil {
				return err
			}

			for _, f := range ff {
				if f.Name == name {
					return config.cloud.UpdatePipelineFile(config.ctx, f.ID, cloud.UpdatePipelineFileOpts{
						Contents:  &contents,
						Encrypted: &encrypt,
					})
				}
			}

			return fmt.Errorf("pipeline file %q not found", name)
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&file, "file", "", "File path. The file you want to update. It must exists already.")
	fs.BoolVar(&encrypt, "encrypt", false, "Encrypt file contents")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
