package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/completer"
	cfg "github.com/calyptia/cli/pkg/config"
)

func newCmdUpdatePipelineFile(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var file string
	var encrypt bool
	completer := completer.Completer{Config: config}

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

			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			ff, err := config.Cloud.PipelineFiles(config.Ctx, pipelineID, cloud.PipelineFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return config.Cloud.UpdatePipelineFile(config.Ctx, f.ID, cloud.UpdatePipelineFile{
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

	_ = cmd.RegisterFlagCompletionFunc("pipeline", completer.CompletePipelines)
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
