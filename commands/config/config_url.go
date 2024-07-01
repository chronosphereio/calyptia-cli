package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	versioncmd "github.com/calyptia/cli/commands/version"
	"github.com/calyptia/cli/config"
)

var ErrURLNotFound = errors.New("url not found")

const KeyBaseURL = "base_url"

func NewCmdConfigSetURL(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "set_url URL",
		Short: "Set the default cloud URL so you don't have to specify it on all commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cloudURL, err := url.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid cloud url: %w", err)
			}

			if cloudURL.Scheme != "http" && cloudURL.Scheme != "https" {
				return fmt.Errorf("invalid cloud url scheme %q", cloudURL.Scheme)
			}

			err = cfg.LocalData.Save(KeyBaseURL, cloudURL.String())
			if err != nil {
				return err
			}

			cfg.BaseURL = cloudURL.String()

			return nil
		},
	}
}

func NewCmdConfigCurrentURL(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_url",
		Short: "Get the current configured default cloud URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println(cfg.BaseURL)
			return nil
		},
	}
}

func NewCmdConfigUnsetURL(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_url",
		Short: "Unset the current configured default cloud URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := cfg.LocalData.Delete(KeyBaseURL)
			if err != nil {
				return err
			}
			cfg.BaseURL = versioncmd.DefaultCloudURLStr
			return nil
		},
	}
}
