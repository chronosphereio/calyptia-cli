package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/cmd/operator"
)

func newCmdInstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install calyptia components",
	}

	cmd.AddCommand(
		operator.NewCmdInstall(),
	)

	return cmd
}
