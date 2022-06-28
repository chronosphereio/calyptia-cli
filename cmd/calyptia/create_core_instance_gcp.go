package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newCmdCreateCoreInstanceOnGCP(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "gcp",
		Aliases: []string{"google", "gce"},
		Short:   "Setup a new core instance on Google Compute Engine (TODO)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
