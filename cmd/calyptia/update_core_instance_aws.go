package main

import (
	"errors"

	"github.com/calyptia/cli/cmd/calyptia/utils"
	"github.com/spf13/cobra"
)

func newCmdUpdateCoreInstanceOnAWS(config *utils.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "aws CORE_INSTANCE",
		Aliases:           []string{"ec2", "amazon"},
		Short:             "Update a core instance from Amazon EC2 (TODO)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.CompleteCoreInstances,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
