package cmd

import (
	"github.com/calyptia/cli/cmd/operator"
	"github.com/spf13/cobra"
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
