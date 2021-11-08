package main

import (
	"encoding/json"
	"errors"
	"fmt"

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

	return keyring.Set(serviceName, "token", string(b))
}

func savedToken() (*oauth2.Token, error) {
	s, err := keyring.Get(serviceName, "token")
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
