package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const serviceName = "cloud.calyptia.com"

var errAccessTokenNotFound = errors.New("access token not found")

func saveAccessToken(tok *oauth2.Token) error {
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("could not json marshall access token: %w", err)
	}

	return keyring.Set(serviceName, "access_token", string(b))
}

func savedAccessToken() (*oauth2.Token, error) {
	s, err := keyring.Get(serviceName, "access_token")
	if err == keyring.ErrNotFound {
		return nil, errAccessTokenNotFound
	}

	if err != nil {
		return nil, err
	}

	var tok *oauth2.Token
	err = json.Unmarshal([]byte(s), &tok)
	if err != nil {
		return nil, fmt.Errorf("could not json unmarshall access token: %w", err)
	}

	return tok, nil
}

func deleteAccessToken() error {
	return keyring.Delete(serviceName, "access_token")
}
