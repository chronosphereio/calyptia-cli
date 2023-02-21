package cmd

import (
	cnfg "github.com/calyptia/cli/cmd/config"
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
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
