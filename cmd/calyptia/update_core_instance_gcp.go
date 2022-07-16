package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newCmdUpdateCoreInstanceOnGCP(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "gcp CORE_INSTANCE",
		Aliases:           []string{"google", "gce"},
		Short:             "Update a core instance from Google Compute Engine (TODO)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAggregators,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
