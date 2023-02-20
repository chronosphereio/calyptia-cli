package main

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
)

var errURLNotFound = errors.New("url not found")

const KeyBaseURL = "base_url"

func newCmdConfigSetURL(config *config) *cobra.Command {
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

			err = config.localData.Save(KeyBaseURL, cloudURL.String())
			if err != nil {
				return err
			}

			config.baseURL = cloudURL.String()

			return nil
		},
	}
}

func newCmdConfigCurrentURL(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_url",
		Short: "Get the current configured default cloud URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println(config.baseURL)
			return nil
		},
	}
}

func newCmdConfigUnsetURL(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_url",
		Short: "Unset the current configured default cloud URL",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.localData.Delete(KeyBaseURL)
			if err != nil {
				return err
			}
			config.baseURL = defaultCloudURLStr
			return nil
		},
	}
}
