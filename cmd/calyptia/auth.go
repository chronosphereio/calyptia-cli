package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/calyptia/cloud-cli/auth0"
	"github.com/zalando/go-keyring"
)

const serviceName = "cloud.calyptia.com"

var errAccessTokenNotFound = errors.New("access token not found")

func saveAccessToken(tok auth0.AccessToken) error {
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("could not json marshall access token: %w", err)
	}

	return keyring.Set(serviceName, "access_token", string(b))
}

func savedAccessToken() (auth0.AccessToken, error) {
	var tok auth0.AccessToken
	s, err := keyring.Get(serviceName, "access_token")
	if err == keyring.ErrNotFound {
		return tok, errAccessTokenNotFound
	}

	if err != nil {
		return tok, err
	}

	err = json.Unmarshal([]byte(s), &tok)
	if err != nil {
		return tok, fmt.Errorf("could not json unmarshall access token: %w", err)
	}

	return tok, nil
}

func deleteAccessToken() error {
	return keyring.Delete(serviceName, "access_token")
}
