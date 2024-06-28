package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/commands/utils"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdUpdatePipelineSecret(config *cfg.Config) *cobra.Command {

	return &cobra.Command{
		Use:               "pipeline_secret ID VALUE",
		Short:             "Update a pipeline secret value",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: config.Completer.CompleteSecretIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// TODO: update secret by its key. The key is unique per pipeline.
			secretID, value := args[0], args[1]
			err := config.Cloud.UpdatePipelineSecret(ctx, secretID, cloud.UpdatePipelineSecret{
				Value: utils.PtrBytes([]byte(value)),
			})
			if err != nil {
				return fmt.Errorf("could not update pipeline secret: %w", err)
			}

			return nil
		},
	}
}
