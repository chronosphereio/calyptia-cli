package commands

import (
	"cmp"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cloudclient "github.com/calyptia/api/client"
	cnfg "github.com/calyptia/cli/commands/config"
	"github.com/calyptia/cli/commands/version"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/localdata"
)

func NewRootCmd() *cobra.Command {
	client := &cloudclient.Client{
		Client: http.DefaultClient,
	}

	storageDir := os.Getenv("CALYPTIA_STORAGE_DIR")
	if storageDir == "" {
		baseDir, err := os.UserHomeDir()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("could not set a base directory for storing local configuration: %w", err))
		}
		storageDir = filepath.Join(baseDir, cnfg.BackUpFolder)
	}

	localData := localdata.New(cnfg.ServiceName, storageDir)
	cfg := &config.Config{
		Cloud:     client,
		LocalData: localData,
		Completer: &completer.Completer{},
	}

	token, err := localData.Get(cnfg.KeyToken)
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
		cfg.BaseURL = client.BaseURL

		if token == "" {
			return
		}

		projectID, err := cnfg.DecodeToken([]byte(token))
		if err != nil {
			cobra.CheckErr(err)
			return
		}

		client.SetProjectToken(token)
		cfg.ProjectToken = token
		cfg.ProjectID = projectID

		cfg.Completer.Cloud = client
		cfg.Completer.ProjectID = projectID
	})
	cmd := &cobra.Command{
		Use:           "calyptia",
		Short:         "Calyptia Cloud CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetOut(os.Stdout)

	fs := cmd.PersistentFlags()
	fs.StringVar(&cloudURLStr, "cloud-url", cmp.Or(os.Getenv("CALYPTIA_CLOUD_URL"), cloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", cmp.Or(os.Getenv("CALYPTIA_CLOUD_TOKEN"), token), "Calyptia Cloud Project token")
	fs.Lookup("token").DefValue = "check with the 'calyptia config current_token' command"

	cmd.AddCommand(
		newCmdConfig(cfg),
		newCmdCreate(cfg),
		newCmdGet(cfg),
		newCmdUpdate(cfg),
		newCmdRollout(cfg),
		newCmdInstall(),
		newCmdUninstall(),
		newCmdDelete(cfg),
		newCmdWatch(cfg),
		version.NewVersionCommand(),
	)

	return cmd
}
