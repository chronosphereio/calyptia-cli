package config

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"

	"github.com/calyptia/cli/cmd/version"
	cfg "github.com/calyptia/cli/config"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var ErrURLNotFound = errors.New("url not found")

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

			err = SaveURL(cloudURL.String())
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
			err := DeleteSavedURL()
			if err != nil {
				return err
			}
			config.BaseURL = version.DefaultCloudURLStr
			return nil
		},
	}
}

func SaveURL(url string) error {
	err := keyring.Set(serviceName, "base_url", url)
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home dir: %w", err)
	}

	fileName := filepath.Join(home, ".calyptia", "base_url")
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		dir := filepath.Dir(fileName)
		err = os.MkdirAll(dir, fs.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create directory %q: %w", dir, err)
		}
	}

	err = os.WriteFile(fileName, []byte(url), fs.ModePerm)
	if err != nil {
		return fmt.Errorf("could not store file %q: %w", fileName, err)
	}

	return nil
}

func DeleteSavedURL() error {
	err := keyring.Delete(serviceName, "base_url")
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := filepath.Join(home, ".calyptia", "base_url")
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil
	}

	err = os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("could not delete default project url: %w", err)
	}

	return nil
}

func SavedURL() (string, error) {
	url, err := keyring.Get(serviceName, "base_url")
	if err == keyring.ErrNotFound {
		return "", ErrURLNotFound
	}

	if err == nil {
		return url, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home dir: %w", err)
	}

	b, err := cfg.ReadFile(filepath.Join(home, ".calyptia", "base_url"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrURLNotFound
	}

	if err != nil {
		return "", err
	}

	url = string(b)

	return url, nil
}