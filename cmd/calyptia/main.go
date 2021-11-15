package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"

	"github.com/calyptia/cloud"
	"github.com/calyptia/cloud-cli/auth0"
	cloudclient "github.com/calyptia/cloud/client"
	"github.com/campoy/unique"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
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

	fs := cmd.Flags()
	fs.StringVar(&config.auth0.Domain, "auth0-domain", env("AUTH0_DOMAIN", "sso.calyptia.com"), "Auth0 domain")
	fs.StringVar(&config.auth0.ClientID, "auth0-client-id", env("AUTH0_CLIENT_ID", defaultAuth0ClientID), "Auth0 client ID")
	fs.StringVar(&config.auth0.Audience, "auth0-audience", env("AUTH0_AUDIENCE", "https://config.calyptia.com"), "Auth0 audience")
	fs.StringVar(&cloudURLStr, "cloud-url", env("CALYPTIA_CLOUD_URL", defaultCloudURLStr), "Calyptia Cloud URL")

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
	cloud *cloudclient.Client
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

func (config *config) completeAgentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.cloud.Projects(config.ctx, 0)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(pp) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var out []string
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, p := range pp {
		p := p
		g.Go(func() error {
			aa, err := config.cloud.Agents(gctx, p.ID, 0)
			if err != nil {
				return err
			}

			mu.Lock()
			for _, a := range aa {
				out = append(out, a.ID)
			}
			mu.Unlock()

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	unique.Slice(&out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (config *config) completeAggregatorIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.fetchAllAggregators()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return aggregatorsKeys(aa), cobra.ShellCompDirectiveNoFileComp
}

// aggregatorsKeys returns unique aggregator names first and then IDs.
func aggregatorsKeys(aa []cloud.Aggregator) []string {
	namesCount := map[string]int{}
	for _, a := range aa {
		if _, ok := namesCount[a.Name]; ok {
			namesCount[a.Name] += 1
			continue
		}

		namesCount[a.Name] = 1
	}

	var out []string

	for _, a := range aa {
		var nameIsUnique bool
		for name, count := range namesCount {
			if a.Name == name && count == 1 {
				nameIsUnique = true
				break
			}
		}
		if nameIsUnique {
			out = append(out, a.Name)
			continue
		}

		out = append(out, a.ID)
	}

	return out
}

func findAggregatorByName(aa []cloud.Aggregator, name string) (cloud.Aggregator, bool) {
	for _, a := range aa {
		if a.Name == name {
			return a, true
		}
	}
	return cloud.Aggregator{}, false
}

func (config *config) completePipelineIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.cloud.Projects(config.ctx, 0)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(pp) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var out []string
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, p := range pp {
		p := p
		g.Go(func() error {
			aa, err := config.cloud.Aggregators(gctx, p.ID, 0)
			if err != nil {
				return err
			}

			g2, gctx2 := errgroup.WithContext(gctx)
			for _, a := range aa {
				a := a
				g2.Go(func() error {
					pp, err := config.cloud.AggregatorPipelines(gctx2, a.ID, 0)
					if err != nil {
						return err
					}

					mu.Lock()
					for _, p := range pp {
						out = append(out, p.ID)
					}
					mu.Unlock()

					return nil
				})
			}
			return g2.Wait()
		})
	}
	if err := g.Wait(); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	unique.Slice(&out, func(i, j int) bool {
		return out[i] < out[j]
	})

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (config *config) completeOutputFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json"}, cobra.ShellCompDirectiveNoFileComp
}

var reUUID4 = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

func validUUID(s string) bool {
	return reUUID4.MatchString(s)
}
