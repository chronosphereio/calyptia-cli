package main

import (
	"fmt"

	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func newCmdUpdateAggregator(config *config) *cobra.Command {
	var newName string

	cmd := &cobra.Command{
		Use:               "aggregator AGGREGATOR",
		Short:             "Update a single aggregator by ID or name",
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

			err = config.cloud.UpdateAggregator(config.ctx, aggregatorID, cloud.UpdateAggregatorOpts{
				Name: &newName,
			})
			if err != nil {
				return fmt.Errorf("could not update aggregator: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&newName, "new-name", "", "New aggregator name")

	_ = cmd.MarkFlagRequired("new-name")

	return cmd
}
