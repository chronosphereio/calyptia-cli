package commands

import (
	"github.com/spf13/cobra"

	cnfg "github.com/calyptia/cli/commands/config"
	cfg "github.com/calyptia/cli/config"
)

func newCmdConfig(config *cfg.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure Calyptia CLI",
	}

	cmd.AddCommand(
		cnfg.NewCmdConfigSetToken(config),
		cnfg.NewCmdConfigCurrentToken(config),
		cnfg.NewCmdConfigUnsetToken(config),
		cnfg.NewCmdConfigSetURL(config),
		cnfg.NewCmdConfigCurrentURL(config),
		cnfg.NewCmdConfigUnsetURL(config),
	)

	return cmd
}
