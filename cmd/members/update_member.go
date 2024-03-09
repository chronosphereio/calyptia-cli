package members

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/calyptia/api/types"

	"github.com/chronosphereio/calyptia-cli/completer"
	"github.com/chronosphereio/calyptia-cli/config"
)

func NewCmdUpdateMember(config *config.Config) *cobra.Command {
	completer := &completer.Completer{Config: config}

	var permissions []string

	cmd := &cobra.Command{
		Use:               "member MEMBER-ID",
		Short:             "Update a member permissions given its membership ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteMembers,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			memberID := args[0]

			in := types.UpdateMember{
				MemberID: memberID,
			}

			if cmd.Flags().Changed("permissions") {
				in.Permissions = &permissions
			}

			// If the user passed "all" as the only permission,
			// we pass an empty slice to the API to grant all permissions.
			if len(permissions) == 1 && permissions[0] == "all" {
				in.Permissions = &[]string{}
			}

			err := config.Cloud.UpdateMember(ctx, in)
			if err != nil {
				return fmt.Errorf("update member: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringSliceVar(&permissions, "permissions", nil, "Permissions to grant to the member")

	_ = cmd.RegisterFlagCompletionFunc("permissions", completer.CompletePermissions)

	return cmd
}
