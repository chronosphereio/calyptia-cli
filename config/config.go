// Package config contains shared dependencies for all commands.
package config

import (
	"github.com/calyptia/api/client"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/localdata"
)

// Config injects the dependencies that all commands share.
type Config struct {
	BaseURL      string
	Cloud        *client.Client
	ProjectToken string
	ProjectID    string
	LocalData    *localdata.Keyring
	Completer    *completer.Completer
}
