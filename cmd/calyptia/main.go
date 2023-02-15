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
	"github.com/calyptia/cli/cmd/calyptia/config"
	"github.com/calyptia/cli/cmd/calyptia/utils"
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
	cfg := &utils.Config{
		Ctx:   ctx,
		Cloud: *client,
	}

	token, err := utils.SavedToken()
	if err != nil && err != utils.ErrTokenNotFound {
		cobra.CheckErr(fmt.Errorf("could not retrive your stored token: %w", err))
	}

	cloudURLStr, err := config.SavedURL()
	if err != nil && err != utils.ErrURLNotFound {
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
		cfg.BaseURL = client.BaseURL

		if token == "" {
			return
		}

		projectID, err := utils.DecodeToken([]byte(token))
		if err != nil {
			return
		}

		client.SetProjectToken(token)
		cfg.ProjectToken = token
		cfg.ProjectID = projectID
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
	fs.StringVar(&cloudURLStr, "cloud-url", utils.Env("CALYPTIA_CLOUD_URL", cloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", utils.Env("CALYPTIA_CLOUD_TOKEN", token), "Calyptia Cloud Project token")

	cmd.AddCommand(
		config.NewCmdConfig(cfg),
		newCmdCreate(cfg),
		newCmdGet(cfg),
		newCmdUpdate(cfg),
		newCmdRollout(cfg),
		newCmdDelete(cfg),
		newCmdTop(cfg),
	)

	return cmd
}
