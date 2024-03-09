package cmd

import (
	"github.com/spf13/cobra"

	cnfg "github.com/chronosphereio/calyptia-cli/cmd/config"
	cfg "github.com/chronosphereio/calyptia-cli/config"
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
