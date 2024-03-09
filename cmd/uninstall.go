package cmd

import (
	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/cmd/operator"
)

func newCmdUninstall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall calyptia components",
	}

	cmd.AddCommand(
		operator.NewCmdUninstall(),
	)

	return cmd
}
