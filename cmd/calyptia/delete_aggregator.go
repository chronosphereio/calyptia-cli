package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newCmdDeleteAggregator(config *config) *cobra.Command {
	var confirmed bool
	cmd := &cobra.Command{
		Use:               "aggregator AGGREGATOR",
		Short:             "Delete a single aggregator by ID or name",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAggregators,
		RunE: func(cmd *cobra.Command, args []string) error {
			aggregatorKey := args[0]
			if !confirmed {
				fmt.Printf("Are you sure you want to delete %q? (y/N) ", aggregatorKey)
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

			aggregatorID := aggregatorKey
			{
				aa, err := config.fetchAllAggregators()
				if err != nil {
					return err
				}

				a, ok := findAggregatorByName(aa, aggregatorKey)
				if !ok && !validUUID(aggregatorID) {
					return nil // idempotent
				}

				if ok {
					aggregatorID = a.ID
				}
			}

			err := config.cloud.DeleteAggregator(config.ctx, aggregatorID)
			if err != nil {
				return fmt.Errorf("could not delete aggregator: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")

	return cmd
}
