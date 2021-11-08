package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/calyptia/cloud-cli/auth0"
	"github.com/calyptia/cloud-cli/cloud"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func main() {
	_ = godotenv.Load()

	cmd := newCmd(context.Background())
	cobra.CheckErr(cmd.Execute())
}

func newCmd(ctx context.Context) *cobra.Command {
	config := &config{
		ctx: ctx,
		auth0: &auth0.Client{
			HTTPClient: http.DefaultClient,
		},
		cloud: &cloud.Client{
			HTTPClient: http.DefaultClient, // Will be replaced by Auth0 token source.
		},
	}

	var cloudURLStr string

	cobra.OnInitialize(func() {
		if config.auth0.ClientID == "" {
			cobra.CheckErr(fmt.Errorf("missing required auth0 client id"))
		}

		cloudURL, err := url.Parse(cloudURLStr)
		if err != nil {
			cobra.CheckErr(fmt.Errorf("invalid cloud url: %w", err))
		}

		if cloudURL.Scheme != "http" && cloudURL.Scheme != "https" {
			cobra.CheckErr(fmt.Errorf("invalid cloud url scheme %q", cloudURL.Scheme))
		}

		config.cloud.BaseURL = cloudURL

		tok, err := savedToken()
		if err == errTokenNotFound {
			return
		}

		if err != nil {
			cobra.CheckErr(fmt.Errorf("could not retrive your stored auth info: %w", err))
		}

		// Now all requests will be authenticated and the token refreshes by its own.
		config.cloud.HTTPClient = config.auth0.Client(config.ctx, tok)
	})
	cmd := &cobra.Command{
		Use:   "calyptia",
		Short: "Calyptia Cloud CLI",
	}

	fs := cmd.Flags()
	fs.StringVar(&config.auth0.Domain, "auth0-domain", env("AUTH0_DOMAIN", "sso.calyptia.com"), "Auth0 domain")
	fs.StringVar(&config.auth0.ClientID, "auth0-client-id", os.Getenv("AUTH0_CLIENT_ID"), "Auth0 client ID")
	fs.StringVar(&config.auth0.Audience, "auth0-audience", env("AUTH0_AUDIENCE", "https://config.calyptia.com"), "Auth0 audience")
	fs.StringVar(&cloudURLStr, "cloud-url", env("CALYPTIA_CLOUD_URL", "https://cloud-api.calyptia.com"), "Calyptia Cloud URL")

	cmd.AddCommand(
		newCmdLogin(config),
		newCmdGet(config),
		newCmdTop(config),
	)

	return cmd
}

type config struct {
	ctx   context.Context
	auth0 *auth0.Client
	cloud *cloud.Client
}

func env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func (config *config) completeProjectIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.cloud.Projects(config.ctx, 0)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(pp) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	out := make([]string, len(pp))
	for i, p := range pp {
		out[i] = p.ID
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (config *config) completeOutputFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
}
