package pipeline

import (
	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
)

func NewCmdDeletePipelineFile(cfg *config.Config) *cobra.Command {
	var confirmed bool
	var pipelineKey string
	var name string

	cmd := &cobra.Command{
		Use:   "pipeline_file",
		Short: "Delete a single file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if !confirmed {
				cmd.Printf("Are you sure you want to delete %q? (y/N) ", pipelineKey)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

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
					return cfg.Cloud.DeletePipelineFile(ctx, f.ID)
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&name, "name", "", "File name you want to delete")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)
	_ = cmd.MarkFlagRequired("name")

	_ = cmd.MarkFlagRequired("pipeline") // TODO: use default pipeline key from config cmd.

	return cmd
}
