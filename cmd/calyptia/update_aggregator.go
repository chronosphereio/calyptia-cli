package main

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdUpdateAggregator(config *config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:               "core_instance CORE_INSTANCE",
		Aliases:           []string{"aggregator"},
		Short:             "Update a single core instance by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAggregators,
		RunE: func(cmd *cobra.Command, args []string) error {
			if newName == "" {
				return nil
			}

			aggregatorKey := args[0]
			// We can only update the aggregator name. Early return if its the same.
			if aggregatorKey == newName {
				return nil
			}

			aggregatorID, err := config.loadAggregatorID(aggregatorKey)
			if err != nil {
				return err
			}

			err = config.cloud.UpdateAggregator(config.ctx, aggregatorID, cloud.UpdateAggregator{
				Name: &newName,
			})
			if err != nil {
				return fmt.Errorf("could not update core instance: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&newName, "new-name", "", "New core instance name")

	_ = cmd.MarkFlagRequired("new-name")

	return cmd
}
