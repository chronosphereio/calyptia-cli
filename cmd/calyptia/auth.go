package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/calyptia/cloud-cli/auth0"
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "cloud.calyptia.com"
	authPrefix  = "auth."
)

func localAccessToken() (string, error) {
	accessToken, err := keyring.Get(serviceName, authPrefix+"access_token")
	if err == keyring.ErrUnsupportedPlatform || err == keyring.ErrNotFound {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("could not retrieve local access token: %w", err)
	}

	expiresInStr, err := keyring.Get(serviceName, authPrefix+"expires_in")
	if err == keyring.ErrUnsupportedPlatform || err == keyring.ErrNotFound {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("could not retrieve local access token expires in: %w", err)
	}

	expiresIn, err := strconv.ParseInt(expiresInStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("could not int parse expires in: %w", err)
	}

	expired := time.Unix(expiresIn, 0).After(time.Now())
	if expired {
		return "", nil
	}

	return accessToken, nil
}

func localRefreshToken() (string, error) {
	refreshToken, err := keyring.Get(serviceName, authPrefix+"refresh_token")
	if err == keyring.ErrUnsupportedPlatform || err == keyring.ErrNotFound {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("could not retrieve local regresh token: %w", err)
	}

	return refreshToken, nil
}

func saveLocalAuth(tok auth0.AccessToken) {
	_ = keyring.Set(serviceName, authPrefix+"access_token", tok.AccessToken)
	_ = keyring.Set(serviceName, authPrefix+"expires_in", strconv.FormatInt(tok.ExpiresIn, 10))
	_ = keyring.Set(serviceName, authPrefix+"refresh_token", tok.RefreshToken)
}

func cleanupLocalAuth() {
	_ = keyring.Delete(serviceName, authPrefix+"access_token")
	_ = keyring.Delete(serviceName, authPrefix+"expires_in")
	_ = keyring.Delete(serviceName, authPrefix+"refresh_token")
}
