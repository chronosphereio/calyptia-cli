package pipeline

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/pointer"
)

func NewCmdDeletePipeline(cfg *config.Config) *cobra.Command {
	var confirmed bool

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Delete a single pipeline by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompletePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineKey := args[0]
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

			err = cfg.Cloud.DeletePipeline(ctx, pipelineID)
			if err != nil {
				return fmt.Errorf("could not delete pipeline: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	return cmd
}

func NewCmdDeletePipelines(cfg *config.Config) *cobra.Command {
	var confirmed bool
	var coreInstanceKey string
	var environmentKey string

	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: "Delete many pipelines from a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var environmentID string
			if environmentKey != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environmentKey)
				if err != nil {
					return err
				}
			}

			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			pp, err := cfg.Cloud.Pipelines(ctx, cloudtypes.PipelinesParams{
				Last:           pointer.From(uint(0)),
				CoreInstanceID: &coreInstanceID,
			})
			if err != nil {
				return fmt.Errorf("could not prefetch pipelines to delete: %w", err)
			}

			if len(pp.Items) == 0 {
				cmd.Println("No pipelines to delete")
				return nil
			}

			if !confirmed {
				cmd.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (y/N) ", strings.Join(completer.PipelinesKeys(pp.Items), "\n"))
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			pipelineIDs := make([]string, len(pp.Items))
			for i, p := range pp.Items {
				pipelineIDs[i] = p.ID
			}

			err = cfg.Cloud.DeletePipelines(ctx, coreInstanceID, pipelineIDs...)
			if err != nil {
				return fmt.Errorf("delete pipelines: %w", err)
			}

			cmd.Printf("Successfully deleted %d pipelines\n", len(pipelineIDs))

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm installation if previous installation found")
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.StringVar(&environmentKey, "environment", "", "Calyptia environment ID or name")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)
	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)

	_ = cmd.MarkFlagRequired("core-instance")

	return cmd
}
