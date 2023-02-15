package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const ServiceName = "cloud.calyptia.com"

var (
	ErrTokenNotFound = errors.New("token not found")
	ErrInvalidToken  = errors.New("invalid token")
	ErrURLNotFound   = errors.New("url not found")
)

func SaveToken(token string) error {
	err := keyring.Set(ServiceName, "project_token", token)
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home dir: %w", err)
	}

	fileName := filepath.Join(home, ".calyptia", "project_token")
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		dir := filepath.Dir(fileName)
		err = os.MkdirAll(dir, fs.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create directory %q: %w", dir, err)
		}
	}

	err = os.WriteFile(fileName, []byte(token), fs.ModePerm)
	if err != nil {
		return fmt.Errorf("could not store file %q: %w", fileName, err)
	}

	return nil
}

func SavedToken() (string, error) {
	token, err := keyring.Get(ServiceName, "project_token")
	if err == keyring.ErrNotFound {
		return "", ErrTokenNotFound
	}

	if err == nil {
		return token, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home dir: %w", err)
	}

	b, err := ReadFile(filepath.Join(home, ".calyptia", "project_token"))
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrTokenNotFound
	}

	if err != nil {
		return "", err
	}

	token = string(b)

	return token, nil
}

func DeleteSavedToken() error {
	err := keyring.Delete(ServiceName, "project_token")
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := filepath.Join(home, ".calyptia", "project_token")
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return nil
	}

	err = os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("could not delete default project token: %w", err)
	}

	return nil
}

type projectTokenPayload struct {
	ProjectID string // no json tag
}

// DecodeToken decodes a project token without verifying its signature
// and getting its inner project ID.
func DecodeToken(token []byte) (string, error) {
	parts := bytes.Split(token, []byte("."))
	if len(parts) != 2 {
		return "", ErrInvalidToken
	}

	encodedPayload := parts[0]

	payload := make([]byte, base64.RawURLEncoding.DecodedLen(len(encodedPayload)))
	n, err := base64.RawURLEncoding.Decode(payload, encodedPayload)
	if err != nil {
		return "", ErrInvalidToken
	}

	payload = payload[:n]

	var out projectTokenPayload
	err = json.Unmarshal(payload, &out)
	if err != nil {
		return "", fmt.Errorf("could not json parse project token payload: %w", err)
	}

	return out.ProjectID, nil
}
