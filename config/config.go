package config

import (
	"github.com/calyptia/api/client"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/localdata"
)

type Config struct {
	BaseURL      string
	Cloud        *client.Client
	ProjectToken string
	ProjectID    string
	LocalData    *localdata.Keyring
	Completer    *completer.Completer
}
