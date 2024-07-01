package commands

import (
	"github.com/spf13/cobra"

	configcmd "github.com/calyptia/cli/commands/config"
	"github.com/calyptia/cli/config"
)

func newCmdConfig(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure Calyptia CLI",
	}

	cmd.AddCommand(
		configcmd.NewCmdConfigSetToken(cfg),
		configcmd.NewCmdConfigCurrentToken(cfg),
		configcmd.NewCmdConfigUnsetToken(cfg),
		configcmd.NewCmdConfigSetURL(cfg),
		configcmd.NewCmdConfigCurrentURL(cfg),
		configcmd.NewCmdConfigUnsetURL(cfg),
	)

	return cmd
}
