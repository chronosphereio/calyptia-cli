package pipeline

import (
	"github.com/spf13/cobra"

	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
)

func NewCmdDeletePipelineLog(config *cfg.Config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:   "pipeline_log PIPELINE_LOGS_ID",
		Short: "Delete a specific pipeline log",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineLogID := args[0]
			if !confirmed {
				cmd.Printf("Are you sure you want to delete %q? (y/N) ", pipelineLogID)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			_, err := config.Cloud.DeletePipelineLog(ctx, pipelineLogID)
			return err
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	return cmd
}
