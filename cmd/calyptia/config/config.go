package config

import (
	"github.com/calyptia/cli/cmd/calyptia/utils"
	"github.com/spf13/cobra"
)

func NewCmdConfig(config *utils.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure Calyptia CLI",
	}

	cmd.AddCommand(
		newCmdConfigSetToken(config),
		newCmdConfigCurrentToken(config),
		newCmdConfigUnsetToken(config),
		newCmdConfigSetURL(config),
		newCmdConfigCurrentURL(config),
		newCmdConfigUnsetURL(config),
	)

	return cmd
}
