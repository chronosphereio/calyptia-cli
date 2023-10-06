package invitation

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdSendInvitation(config *config.Config) *cobra.Command {
	var redirectURI string

	if config.BaseURL == "https://cloud-api-dev.calyptia.com" {
		redirectURI = "https://core-dev.calyptia.com"
	}
	if config.BaseURL == "https://cloud-api-staging.calyptia.com" {
		redirectURI = "https://core-staging.calyptia.com"
	}

	cmd := &cobra.Command{
		Use:   "invitation EMAIL", // child of `calyptia create`
		Short: "Send an invitation to a user to join the current project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			for _, email := range args {
				err := config.Cloud.CreateInvitation(ctx, config.ProjectID, types.CreateInvitation{
					Email:       email,
					RedirectURI: redirectURI,
				})
				if err != nil {
					cmd.PrintErrf("failed to send invitation for %q: %v\n", email, err)
					continue
				}

				cmd.Printf("invitation sent to %q successfully\n", email)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&redirectURI, "redirect-uri", "https://core.calyptia.com", "Redirect URI for the invitation, leave the default value if you don't know what it is")

	return cmd
}
