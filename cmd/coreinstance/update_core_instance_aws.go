package coreinstance

import (
	"errors"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"

	"github.com/spf13/cobra"
)

func NewCmdUpdateCoreInstanceOnAWS(config *cfg.Config) *cobra.Command {
	completer := completer.Completer{Config: config}
	cmd := &cobra.Command{
		Use:               "aws CORE_INSTANCE",
		Aliases:           []string{"ec2", "amazon"},
		Short:             "Update a core instance from Amazon EC2 (TODO)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteCoreInstances,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
