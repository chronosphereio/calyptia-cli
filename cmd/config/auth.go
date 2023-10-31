package config

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	ServiceName  = "cloud.calyptia.com"
	BackUpFolder = ".calyptia"
)

var ErrInvalidToken = errors.New("invalid token")

type projectTokenPayload struct {
	ProjectID string // no json tag
}

// decodeToken decodes a project token without verifying its signature
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
