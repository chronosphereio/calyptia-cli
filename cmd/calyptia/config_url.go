package main

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"io/fs"
	"os"
	"path/filepath"
)

var errUrlNotFound = errors.New("url not found")

func newCmdConfigSetURL(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "set_url URL",
		Short: "Set the default project url so you don't have to specify it on all commands",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			err := saveUrl(url)
			if err != nil {
				return err
			}
			config.baseURL = url
			return nil
		},
	}
}

func newCmdConfigCurrentURL(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_url",
		Short: "Get the current configured default project url",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), config.baseURL)
			return nil
		},
	}
}

func newCmdConfigUnsetURL(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_url",
		Short: "Unset the current configured default project url",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := deleteSavedUrl()
			if err != nil {
				return err
			}
			config.baseURL = defaultCloudURLStr
			return nil
		},
	}
}

func saveUrl(url string) error {
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

func deleteSavedUrl() error {
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

func savedUrl() (string, error) {
	url, err := keyring.Get(serviceName, "base_url")
	if err == keyring.ErrNotFound {
		return "", errUrlNotFound
	}

	if err == nil {
		return url, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home dir: %w", err)
	}

	b, err := readFile(filepath.Join(home, ".calyptia", "base_url"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", errUrlNotFound
	}

	if err != nil {
		return "", err
	}

	url = string(b)

	return url, nil
}
