package commands

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/commands/operator"
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
