package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	cloudclient "github.com/calyptia/api/client"
	cnfg "github.com/calyptia/cli/cmd/config"
	cliversion "github.com/calyptia/cli/cmd/version"
	utils "github.com/calyptia/cli/cmd/version"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/localdata"
)

var vercheck bool

func NewRootCmd(ctx context.Context) *cobra.Command {
	client := &cloudclient.Client{
		Client: http.DefaultClient,
	}

	_, found := os.LookupEnv("CALYPTIA_DISABLE_VERSION_CHECK") // if environment variable CALYPTIA_DISABLE_VERSION_CHECK is present just disable version check
	vercheck = !found

	storageDir := os.Getenv("CALYPTIA_STORAGE_DIR")
	if storageDir == "" {
		baseDir, err := os.UserHomeDir()
		if err != nil {
			cobra.CheckErr(fmt.Errorf("could not set a base directory for storing local configuration: %w", err))
		}
		storageDir = filepath.Join(baseDir, cnfg.BackUpFolder)
	}

	localData := localdata.New(cnfg.ServiceName, storageDir)
	config := &cfg.Config{
		Ctx:       ctx,
		Cloud:     client,
		LocalData: localData,
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
		cloudURLStr = cliversion.DefaultCloudURLStr
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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if !vercheck {
				versionCheck(cmd)
			}
		},
	}

	cmd.SetOut(os.Stdout)

	fs := cmd.PersistentFlags()
	fs.StringVar(&cloudURLStr, "cloud-url", cfg.Env("CALYPTIA_CLOUD_URL", cloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", cfg.Env("CALYPTIA_CLOUD_TOKEN", token), "Calyptia Cloud Project token")
	fs.BoolVar(&vercheck, "disable-version-check", false, "disable version check ")
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
		newCmdWatch(config),
		cliversion.NewVersionCommand(),
	)

	return cmd
}

func versionCheck(cmd *cobra.Command) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		if len(cliversion.Version) != 0 && cliversion.Version != "dev" {
			ghclient, err := utils.NewGithubClient("", nil)
			if err != nil {
				return
			}

			ref, err := ghclient.GetLatest("cli")
			if err != nil {
				return
			}

			latestVersion, err := version.NewVersion(ref)
			if err != nil {
				return
			}

			currentVersion, err := version.NewVersion(cliversion.Version)
			if err != nil {
				fmt.Println("err", err)
				return
			}

			if currentVersion != nil && currentVersion.LessThan(latestVersion) {
				fmt.Printf("Warning: This version %s of Calyptia cli is outdated. The latest version available is %s\n", currentVersion, latestVersion)
				return
			}
		}
	}
}
