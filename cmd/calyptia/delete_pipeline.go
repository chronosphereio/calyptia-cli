package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCmdDeletePipeline(config *config) *cobra.Command {
	var confirmed bool
	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Delete a single pipeline by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			if !confirmed {
				fmt.Printf("Are you sure you want to delete %q? (y/N) ", pipelineKey)
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

			pipelineID := pipelineKey
			if !validUUID(pipelineID) {
				aa, err := config.fetchAllPipelines()
				if err != nil {
					return err
				}

				a, ok := findPipelineByName(aa, pipelineKey)
				if !ok {
					return nil
				}

				pipelineID = a.ID
			}

			err := config.cloud.DeleteAggregatorPipeline(config.ctx, pipelineID)
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
