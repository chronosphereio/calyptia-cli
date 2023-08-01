package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cloudclient "github.com/calyptia/api/client"
	cnfg "github.com/calyptia/cli/cmd/config"
	"github.com/calyptia/cli/cmd/top"
	"github.com/calyptia/cli/cmd/version"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/localdata"
)

func NewRootCmd(ctx context.Context) *cobra.Command {
	client := &cloudclient.Client{
		Client: http.DefaultClient,
	}

	baseDir, err := os.UserHomeDir()
	if err != nil {
		// no home dir is available, lets fallback to the current working directory
		// if fails to get current working directory, we are on a failure scenario.
		baseDir, err = os.Getwd()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("could not set a base directory for storing local configuration: %w", err))
		}
	}

	localData := localdata.New(cnfg.ServiceName, filepath.Join(baseDir, cnfg.BackUpFolder))
	config := &cfg.Config{
		Ctx:       ctx,
		Cloud:     client,
		LocalData: localData,
	}

	token, err := localData.Get(cnfg.KeyToken)
	// if no data is found (which is okay, the default token will be tried from the parameter --token)
	if err != nil && !errors.Is(err, localdata.ErrNotFound) {
		cobra.CheckErr(fmt.Errorf("could not retrieve your stored token: %w", err))
	}

	cloudURLStr, err := localData.Get(cnfg.KeyBaseURL)
	if err != nil && !errors.Is(err, localdata.ErrNotFound) {
		cobra.CheckErr(fmt.Errorf("could not retrieve your stored cloud url: %w", err))
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
			cobra.CheckErr(err)
			return
		}

		client.SetProjectToken(token)
		config.ProjectToken = token
		config.ProjectID = projectID
	})
	cmd := &cobra.Command{
		Use:           "calyptia",
		Short:         "Calyptia Cloud CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetOut(os.Stdout)

	fs := cmd.PersistentFlags()
	fs.StringVar(&cloudURLStr, "cloud-url", cfg.Env("CALYPTIA_CLOUD_URL", cloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", cfg.Env("CALYPTIA_CLOUD_TOKEN", token), "Calyptia Cloud Project token")
	fs.Lookup("token").DefValue = "check with the 'calyptia config current_token' command"

	cmd.AddCommand(
		newCmdConfig(config),
		newCmdCreate(config),
		newCmdGet(config),
		newCmdUpdate(config),
		newCmdRollout(config),
		newCmdInstall(),
		newCmdUninstall(),
		newCmdDelete(config),
		top.NewCmdTop(config),
		version.NewVersionCommand(),
	)

	return cmd
}
