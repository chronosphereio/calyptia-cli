package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newCmdCreateCoreInstanceOnAWS(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "aws",
		Aliases: []string{"ec2", "amazon"},
		Short:   "Setup a new core instance on Amazon EC2 (TODO)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
