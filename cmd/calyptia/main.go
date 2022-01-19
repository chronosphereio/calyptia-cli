package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"

	"github.com/calyptia/cloud"
	"github.com/calyptia/cloud-cli/auth0"
	cloudclient "github.com/calyptia/cloud/client"
	cloudtoken "github.com/calyptia/cloud/token"
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

	token, err := savedToken()
	if err != nil && err != errTokenNotFound {
		cobra.CheckErr(fmt.Errorf("could not retrive your stored token: %w", err))
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

		config.cloud.BaseURL = cloudclient.Endpoint(cloudURL.String())

		if token == "" {
			return
		}

		tokenVerifier := &cloudtoken.SignVerifier{}
		b, err := tokenVerifier.Decode([]byte(token))
		if err != nil {
			return
		}

		var payload cloud.ProjectTokenPayload
		err = json.Unmarshal(b, &payload)
		if err != nil {
			cobra.CheckErr(fmt.Errorf("could not decode token payload: %w", err))
		}

		config.cloud.SetProjectToken(token)
		config.projectToken = token
		config.projectID = payload.ProjectID
	})
	cmd := &cobra.Command{
		Use:           "calyptia",
		Short:         "Calyptia Cloud CLI",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	fs := cmd.PersistentFlags()
	fs.StringVar(&config.auth0.Domain, "auth0-domain", env("CALYPTIA_AUTH0_DOMAIN", "sso.calyptia.com"), "Auth0 domain")
	fs.StringVar(&config.auth0.ClientID, "auth0-client-id", env("CALYPTIA_AUTH0_CLIENT_ID", defaultAuth0ClientID), "Auth0 client ID")
	fs.StringVar(&config.auth0.Audience, "auth0-audience", env("CALYPTIA_AUTH0_AUDIENCE", "https://config.calyptia.com"), "Auth0 audience")
	fs.StringVar(&cloudURLStr, "cloud-url", env("CALYPTIA_CLOUD_URL", defaultCloudURLStr), "Calyptia Cloud URL")
	fs.StringVar(&token, "token", token, "Project token")

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

type config struct {
	ctx          context.Context
	auth0        *auth0.Client
	cloud        *cloudclient.Client
	projectToken string
	projectID    string
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
