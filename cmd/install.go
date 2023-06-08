package cmd

import (
	"github.com/calyptia/cli/cmd/operator"
	"github.com/spf13/cobra"

	cfg "github.com/calyptia/cli/config"
)

func newCmdInstall(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install calyptia components",
	}

	cmd.AddCommand(
		operator.NewCmdInstall(config, nil),
	)

	return cmd
}
