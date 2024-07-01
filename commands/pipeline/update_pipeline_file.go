package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdatePipelineFile(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var file string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Update a file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			contents, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			name := filepath.Base(file)
			name = strings.TrimSuffix(name, filepath.Ext(name))

			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.PipelineFiles(ctx, pipelineID, cloudtypes.PipelineFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return cfg.Cloud.UpdatePipelineFile(ctx, f.ID, cloudtypes.UpdatePipelineFile{
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

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
