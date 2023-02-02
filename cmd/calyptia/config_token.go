package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

const KeyToken = "project_token"

func newCmdConfigSetToken(config *config) *cobra.Command {
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

			return config.localData.Save(KeyToken, token)
		},
	}
}

func newCmdConfigCurrentToken(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_token",
		Short: "Get the current configured default project token",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.projectToken)
			return nil
		},
	}
}

func newCmdConfigUnsetToken(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_token",
		Short: "Unset the current configured default project token",
		RunE: func(cmd *cobra.Command, args []string) error {
			return config.localData.Delete(KeyToken)
		},
	}
}
