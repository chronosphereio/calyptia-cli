package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newCmdConfigSetProject(config *config) *cobra.Command {
	return &cobra.Command{
		Use:               "set_project PROJECT",
		Short:             "Set the default project so you don't have to specify it on all commands",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := args[0]
			projectID := projectKey
			{
				pp, err := config.cloud.Projects(config.ctx, 0)
				if err != nil {
					return err
				}

				a, ok := findProjectByName(pp, projectKey)
				if !ok && !validUUID(projectID) {
					return fmt.Errorf("could not find project %q", projectKey)
				}

				if ok {
					projectID = a.ID
				}
			}

			if err := saveDefaultProject(projectID); err != nil {
				return err
			}

			config.defaultProject = projectID

			return nil
		},
	}
}

func newCmdConfigCurrentProject(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "current_project",
		Short: "Get the current configured default project",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(config.defaultProject)
			return nil
		},
	}
}

func newCmdConfigUnsetProject(config *config) *cobra.Command {
	return &cobra.Command{
		Use:   "unset_project",
		Short: "Unset the current configured default project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deleteSavedDefaultProject(); err != nil {
				return err
			}

			config.defaultProject = ""
			return nil
		},
	}
}

func saveDefaultProject(projectKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := filepath.Join(home, ".calyptia", "default_project")
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(fileName), fs.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create directory: %w", err)
		}
	}

	err = os.WriteFile(fileName, []byte(projectKey), fs.ModePerm)
	if err != nil {
		return fmt.Errorf("could not store default project: %w", err)
	}

	return nil
}

var errDefaultProjectNotFound = errors.New("default project not found")

func savedDefaultProject() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home dir: %w", err)
	}

	b, err := readFile(filepath.Join(home, ".calyptia", "default_project"))
	if os.IsNotExist(err) || errors.Is(err, fs.ErrNotExist) {
		return "", errDefaultProjectNotFound
	}

	if err != nil {
		return "", err
	}

	projectKey := strings.TrimSpace(string(b))
	return projectKey, nil
}

func deleteSavedDefaultProject() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := filepath.Join(home, ".calyptia", "default_project")
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil
	}

	err = os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("could not delete default project: %w", err)
	}

	return nil
}
