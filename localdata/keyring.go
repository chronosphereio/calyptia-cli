// Package localdata provides a keyring implementation that stores data in the system keyring.
// If the keyring is not available, it falls back to storing the data in a file.
package localdata

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	kr "github.com/zalando/go-keyring"
)

// ErrNotFound is the expected error if the data isn't found in the keyring or in the file.
var ErrNotFound = errors.New("data not found")

type Keyring struct {
	serviceName string
	backupFile  string
}

// New creates a new keyring.
// serviceName is the name of the service that will be used to store the data.
// backupFile is the name of the file that will be used to store the data if the keyring is not available.
// The file is stored in the user's home directory, in a file named after the key.
func New(serviceName, backupFile string) *Keyring {
	return &Keyring{serviceName: serviceName, backupFile: backupFile}
}

// Save stores the data in the keyring.
// If the keyring is not available, it falls back to storing the data in a file.
// The file is stored in the user's home directory, in a file named after the key.
func (k *Keyring) Save(key, data string) error {
	err := kr.Set(k.serviceName, key, data)
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not get user home dir: %w", err)
	}

	fileName := filepath.Join(home, k.backupFile, key)
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		dir := filepath.Dir(fileName)
		err = os.MkdirAll(dir, fs.ModePerm)
		if err != nil {
			return fmt.Errorf("could not create directory %q: %w", dir, err)
		}
	}

	err = os.WriteFile(fileName, []byte(data), fs.ModePerm)
	if err != nil {
		return fmt.Errorf("could not store file %q: %w", fileName, err)
	}

	return nil
}

// Get retrieves the data from the keyring.
// If the keyring is not available, it falls back to retrieving the data from a file.
func (k *Keyring) Get(key string) (string, error) {
	data, err := kr.Get(k.serviceName, key)
	if err == kr.ErrNotFound {
		return "", ErrNotFound
	}

	if err == nil {
		return data, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home dir: %w", err)
	}

	b, err := readFile(filepath.Join(home, k.backupFile, key))
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNotFound
	}

	if err != nil {
		return "", err
	}

	data = string(b)

	return data, nil
}

// Delete removes the data from the keyring.
// If the keyring is not available, it falls back to removing the data from a file.
func (k *Keyring) Delete(key string) error {
	err := kr.Delete(k.serviceName, key)
	if err == nil {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fileName := filepath.Join(home, k.backupFile, key)
	_, err = os.Stat(fileName)

	if errors.Is(err, fs.ErrNotExist) {
		return ErrNotFound
	}

	err = os.Remove(fileName)
	if err != nil {
		return fmt.Errorf("could not remove file %q: %w", fileName, err)
	}

	return nil
}

func readFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("could not close file: %v", err)
		}
	}(f)

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read contents: %w", err)
	}

	return b, nil
}
