package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const serviceName = "cloud.calyptia.com"

var errTokenNotFound = errors.New("token not found")

func saveToken(tok *oauth2.Token) error {
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("could not json marshall token: %w", err)
	}

	err = keyring.Set(serviceName, "token", string(b))
	if err == keyring.ErrUnsupportedPlatform || errors.Is(err, exec.ErrNotFound) || (err != nil && strings.Contains(err.Error(), "failed to unlock correct collection")) {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		fileName := filepath.Join(home, ".calyptia", "creds")
		if _, err := os.Stat(fileName); os.IsNotExist(err) {
			err = os.MkdirAll(filepath.Dir(fileName), fs.ModePerm)
			if err != nil {
				return fmt.Errorf("could not create directory: %w", err)
			}
		}

		err = os.WriteFile(fileName, b, fs.ModePerm)
		if err != nil {
			return fmt.Errorf("could not store creds: %w", err)
		}

		return nil
	}

	if err != nil {
		return err
	}

	return nil
}

func savedToken() (*oauth2.Token, error) {
	s, err := keyring.Get(serviceName, "token")
	if err == keyring.ErrUnsupportedPlatform || errors.Is(err, exec.ErrNotFound) || (err != nil && strings.Contains(err.Error(), "failed to unlock correct collection")) {
		err = nil

		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not get user home dir: %w", err)
		}

		b, err := readFile(filepath.Join(home, ".calyptia", "creds"))
		if os.IsNotExist(err) || errors.Is(err, fs.ErrNotExist) {
			return nil, errTokenNotFound
		}

		if err != nil {
			return nil, err
		}

		s = string(b)
	}

	if err == keyring.ErrNotFound {
		return nil, errTokenNotFound
	}

	if err != nil {
		return nil, err
	}

	var tok *oauth2.Token
	err = json.Unmarshal([]byte(s), &tok)
	if err != nil {
		return nil, fmt.Errorf("could not json unmarshall token: %w", err)
	}

	return tok, nil
}
