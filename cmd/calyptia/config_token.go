package main

import (
	"fmt"

	cfg "github.com/calyptia/cli/pkg/config"
	"github.com/spf13/cobra"
)

func newCmdConfigSetToken(config *cfg.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "set_token TOKEN",
		Short: "Set the default project token so you don't have to specify it on all commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token := args[0]
			_, err := decodeToken([]byte(token))
			if err != nil {
				return err
			}

			return saveToken(token)
		},
	}
}

func newCmdConfigCurrentToken(config *cfg.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_token",
		Short: "Get the current configured default project token",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.ProjectToken)
			return nil
		},
	}
}

func newCmdConfigUnsetToken(config *cfg.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_token",
		Short: "Unset the current configured default project token",
		RunE: func(cmd *cobra.Command, args []string) error {
			return deleteSavedToken()
		},
	}
}
