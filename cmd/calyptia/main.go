package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/calyptia/cloud-cli/auth0"
	cloudclient "github.com/calyptia/cloud/client"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	defaultAuth0ClientID string // To be injected at build time: -ldflags="-X 'main.defaultAuth0ClientID=xxx'"
	defaultCloudURLStr   = "https://cloud-api.calyptia.com"
	version              = "dev" // To be injected at build time: -ldflags="-X 'main.version=xxx'"
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
		cloud: &cloudclient.Client{
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

		config.cloud.BaseURL = cloudURL.String()

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
		Use:     "calyptia",
		Short:   "Calyptia Cloud CLI",
		Version: version,
	}

	fs := cmd.PersistentFlags()
	fs.StringVar(&config.auth0.Domain, "auth0-domain", env("AUTH0_DOMAIN", "sso.calyptia.com"), "Auth0 domain")
	fs.StringVar(&config.auth0.ClientID, "auth0-client-id", env("AUTH0_CLIENT_ID", defaultAuth0ClientID), "Auth0 client ID")
	fs.StringVar(&config.auth0.Audience, "auth0-audience", env("AUTH0_AUDIENCE", "https://config.calyptia.com"), "Auth0 audience")
	fs.StringVar(&cloudURLStr, "cloud-url", env("CALYPTIA_CLOUD_URL", defaultCloudURLStr), "Calyptia Cloud URL")

	cmd.AddCommand(
		newCmdLogin(config),
		newCmdCreate(config),
		newCmdGet(config),
		newCmdUpdate(config),
		newCmdDelete(config),
		newCmdTop(config),
	)

	return cmd
}

type config struct {
	ctx   context.Context
	auth0 *auth0.Client
	cloud *cloudclient.Client
}

func env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func (config *config) completeOutputFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
}

var reUUID4 = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

func validUUID(s string) bool {
	return reUUID4.MatchString(s)
}
