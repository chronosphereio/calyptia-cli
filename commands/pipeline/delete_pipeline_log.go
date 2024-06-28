package pipeline

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cfg "github.com/calyptia/cli/config"
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

			_, err := config.Cloud.DeletePipelineLog(ctx, pipelineLogID)
			return err
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	return cmd
}
