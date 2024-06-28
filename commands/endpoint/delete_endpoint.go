package endpoint

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
)

func NewCmdDeleteEndpoint(cfg *config.Config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:               "endpoint ENDPOINT",
		Short:             "Delete a single endpoint by ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompletePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			portID := args[0]
			if !confirmed {
				cmd.Printf("Are you sure you want to delete %q? (y/N) ", portID)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			err := cfg.Cloud.DeletePipelinePort(ctx, portID)
			if err != nil {
				return fmt.Errorf("could not delete endpoint: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	return cmd
}
