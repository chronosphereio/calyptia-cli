package cmd

import (
	"github.com/calyptia/cli/cmd/operator"
	"github.com/spf13/cobra"
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
