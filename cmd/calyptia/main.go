package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	cloudclient "github.com/calyptia/api/client"
	cfg "github.com/calyptia/cli/pkg/config"
)

var (
	defaultCloudURLStr = "https://cloud-api.calyptia.com"
	version            = "dev" // To be injected at build time: -ldflags="-X 'main.version=xxx'"
)

func main() {
	_ = godotenv.Load()

	cmd := newCmd(context.Background())
	cobra.CheckErr(cmd.Execute())
}

func newCmd(ctx context.Context) *cobra.Command {
	client := &cloudclient.Client{
		Client: http.DefaultClient,
	}
	config := &cfg.Config{
		Ctx:   ctx,
		Cloud: client,
	}

	token, err := savedToken()
	if err != nil && err != errTokenNotFound {
		cobra.CheckErr(fmt.Errorf("could not retrive your stored token: %w", err))
	}

	cloudURLStr, err := savedURL()
	if err != nil && err != errURLNotFound {
		cobra.CheckErr(fmt.Errorf("could not retrive your stored url: %w", err))
	}

	if cloudURLStr == "" {
		cloudURLStr = defaultCloudURLStr
	}

	cobra.OnInitialize(func() {
		cloudURL, err := url.Parse(cloudURLStr)
		if err != nil {
			cobra.CheckErr(fmt.Errorf("invalid cloud url: %w", err))
		}

		if cloudURL.Scheme != "http" && cloudURL.Scheme != "https" {
			cobra.CheckErr(fmt.Errorf("invalid cloud url scheme %q", cloudURL.Scheme))
		}

		client.BaseURL = cloudURL.String()
		config.BaseURL = client.BaseURL

		if token == "" {
			return
		}

		projectID, err := decodeToken([]byte(token))
		if err != nil {
			return
		}

		client.SetProjectToken(token)
		config.ProjectToken = token
		config.ProjectID = projectID
	})
	cmd := &cobra.Command{
		Use:           "calyptia",
		Short:         "Calyptia Cloud CLI",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetOut(os.Stdout)

	fs := cmd.PersistentFlags()
	fs.StringVar(&cloudURLStr, "cloud-url", env("CALYPTIA_CLOUD_URL", cloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", env("CALYPTIA_CLOUD_TOKEN", token), "Calyptia Cloud Project token")

	cmd.AddCommand(
		newCmdConfig(config),
		newCmdCreate(config),
		newCmdGet(config),
		newCmdUpdate(config),
		newCmdRollout(config),
		newCmdDelete(config),
		newCmdTop(config),
	)

	return cmd
}

func env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
