package invitation

import (
	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdSendInvitation(cfg *config.Config) *cobra.Command {
	redirectURI := "https://core.calyptia.com"

	if cfg.BaseURL == "https://cloud-api-dev.calyptia.com" {
		redirectURI = "https://core-dev.calyptia.com"
	}
	if cfg.BaseURL == "https://cloud-api-staging.calyptia.com" {
		redirectURI = "https://core-staging.calyptia.com"
	}

	var permissions []string

	cmd := &cobra.Command{
		Use:   "invitation EMAIL", // child of `calyptia create`
		Short: "Send an invitation to a user to join the current project",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			for _, email := range args {
				err := cfg.Cloud.CreateInvitation(ctx, cfg.ProjectID, cloudtypes.CreateInvitation{
					Email:       email,
					RedirectURI: redirectURI,
					Permissions: permissions,
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
	fs.StringVar(&redirectURI, "redirect-uri", redirectURI, "Redirect URI for the invitation, it should point to the Calyptia UI.")
	fs.StringSliceVar(&permissions, "permissions", []string{cloudtypes.PermReadAll}, "Permissions to grant to the invited user.")

	_ = cmd.RegisterFlagCompletionFunc("permissions", cfg.Completer.CompletePermissions)

	return cmd
}
