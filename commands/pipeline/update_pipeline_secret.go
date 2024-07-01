package pipeline

import (
	"fmt"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/pointer"
)

func NewCmdUpdatePipelineSecret(cfg *config.Config) *cobra.Command {

	return &cobra.Command{
		Use:               "pipeline_secret ID VALUE",
		Short:             "Update a pipeline secret value",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: cfg.Completer.CompleteSecretIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			// TODO: update secret by its key. The key is unique per pipeline.
			secretID, value := args[0], args[1]
			err := cfg.Cloud.UpdatePipelineSecret(ctx, secretID, cloudtypes.UpdatePipelineSecret{
				Value: pointer.From([]byte(value)),
			})
			if err != nil {
				return fmt.Errorf("could not update pipeline secret: %w", err)
			}

			return nil
		},
	}
}
