package members

import (
	"fmt"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdateMember(cfg *config.Config) *cobra.Command {
	var permissions []string

	cmd := &cobra.Command{
		Use:               "member MEMBER-ID",
		Short:             "Update a member permissions given its membership ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteMembers,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			memberID := args[0]

			in := cloudtypes.UpdateMember{
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

			err := cfg.Cloud.UpdateMember(ctx, in)
			if err != nil {
				return fmt.Errorf("update member: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringSliceVar(&permissions, "permissions", []string{cloudtypes.PermReadAll}, "Permissions to grant to the member")

	_ = cmd.RegisterFlagCompletionFunc("permissions", cfg.Completer.CompletePermissions)

	return cmd
}
