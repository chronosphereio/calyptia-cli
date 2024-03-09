package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/cmd/version"
	cfg "github.com/chronosphereio/calyptia-cli/config"
)

var ErrURLNotFound = errors.New("url not found")

const KeyBaseURL = "base_url"

func NewCmdConfigSetURL(config *cfg.Config) *cobra.Command {
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

			err = config.LocalData.Save(KeyBaseURL, cloudURL.String())
			if err != nil {
				return err
			}

			config.BaseURL = cloudURL.String()

			return nil
		},
	}
}

func NewCmdConfigCurrentURL(config *cfg.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_url",
		Short: "Get the current configured default cloud URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println(config.BaseURL)
			return nil
		},
	}
}

func NewCmdConfigUnsetURL(config *cfg.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_url",
		Short: "Unset the current configured default cloud URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.LocalData.Delete(KeyBaseURL)
			if err != nil {
				return err
			}
			config.BaseURL = version.DefaultCloudURLStr
			return nil
		},
	}
}
