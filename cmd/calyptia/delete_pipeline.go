package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/calyptia/utils"
)

func newCmdDeletePipeline(config *utils.Config) *cobra.Command {
	var confirmed bool
	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Delete a single pipeline by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.CompletePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			if !confirmed {
				cmd.Printf("Are you sure you want to delete %q? (y/N) ", pipelineKey)
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

			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			err = config.Cloud.DeletePipeline(config.Ctx, pipelineID)
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

func newCmdDeletePipelines(config *utils.Config) *cobra.Command {
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
				environmentID, err = config.LoadEnvironmentID(environmentKey)
				if err != nil {
					return err
				}
			}

			coreInstanceID, err := config.LoadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			pp, err := config.Cloud.Pipelines(ctx, coreInstanceID, types.PipelinesParams{
				Last: utils.Ptr(uint(0)),
			})
			if err != nil {
				return fmt.Errorf("could not prefetch pipelines to delete: %w", err)
			}

			if len(pp.Items) == 0 {
				cmd.Println("No pipelines to delete")
				return nil
			}

			if !confirmed {
				cmd.Printf("You are about to delete:\n\n%s\n\nAre you sure you want to delete all of them? (y/N) ", strings.Join(utils.PipelinesKeys(pp.Items), "\n"))
				confirmed, err := readConfirm(cmd.InOrStdin())
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

			err = config.Cloud.DeletePipelines(ctx, coreInstanceID, pipelineIDs...)
			if err != nil {
				return fmt.Errorf("delete pipelines: %w", err)
			}

			cmd.Printf("Successfully deleted %d pipelines\n", len(pipelineIDs))

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Parent core-instance ID or name")
	fs.StringVar(&environmentKey, "environment", "", "Calyptia environment ID or name")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", config.CompleteCoreInstances)
	_ = cmd.RegisterFlagCompletionFunc("environment", config.CompleteEnvironments)

	_ = cmd.MarkFlagRequired("core-instance")

	return cmd
}
