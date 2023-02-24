package coreinstance

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdUpdateCoreInstanceOnGCP(config *cfg.Config) *cobra.Command {
	completer := completer.Completer{Config: config}
	cmd := &cobra.Command{
		Use:               "gcp CORE_INSTANCE",
		Aliases:           []string{"google", "gce"},
		Short:             "Update a core instance from Google Compute Engine (TODO)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteCoreInstances,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("not implemented")
		},
	}
	return cmd
}
