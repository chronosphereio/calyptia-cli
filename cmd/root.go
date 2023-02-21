package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	cloudclient "github.com/calyptia/api/client"
	cnfg "github.com/calyptia/cli/cmd/config"
	"github.com/calyptia/cli/cmd/top"
	"github.com/calyptia/cli/cmd/version"
	cfg "github.com/calyptia/cli/config"
)

func NewRootCmd(ctx context.Context) *cobra.Command {
	client := &cloudclient.Client{
		Client: http.DefaultClient,
	}
	config := &cfg.Config{
		Ctx:   ctx,
		Cloud: client,
	}

	token, err := cnfg.SavedToken()
	if err != nil && err != cnfg.ErrTokenNotFound {
		cobra.CheckErr(fmt.Errorf("could not retrive your stored token: %w", err))
	}

	cloudURLStr, err := cnfg.SavedURL()
	if err != nil && err != cnfg.ErrURLNotFound {
		cobra.CheckErr(fmt.Errorf("could not retrive your stored url: %w", err))
	}

	if cloudURLStr == "" {
		cloudURLStr = version.DefaultCloudURLStr
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

		projectID, err := cnfg.DecodeToken([]byte(token))
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
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.SetOut(os.Stdout)

	fs := cmd.PersistentFlags()
	fs.StringVar(&cloudURLStr, "cloud-url", cfg.Env("CALYPTIA_CLOUD_URL", cloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", cfg.Env("CALYPTIA_CLOUD_TOKEN", token), "Calyptia Cloud Project token")

	cmd.AddCommand(
		newCmdConfig(config),
		newCmdCreate(config),
		newCmdGet(config),
		newCmdUpdate(config),
		newCmdRollout(config),
		newCmdDelete(config),
		top.NewCmdTop(config),
	)

	return cmd
}
