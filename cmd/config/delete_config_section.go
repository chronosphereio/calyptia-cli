package config

import (
	"fmt"
	"os"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewCmdDeleteConfigSection(config *cfg.Config) *cobra.Command {
	var confirmed bool
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "config_section CONFIG_SECTION", // child of `delete`
		Short:             "Delete config section",
		Long:              "Delete a config section by either the plugin kind:name or ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteConfigSections,
		RunE: func(cmd *cobra.Command, args []string) error {
			configSectionKey := args[0]

			if !confirmed {
				cmd.Printf("Are you sure you want to delete config section %q? (y/N) ", configSectionKey)
				ok, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !ok {
					cmd.Println("Aborted")
					return nil
				}
			}

			ctx := cmd.Context()
			configSectionID, err := completer.LoadConfigSectionID(ctx, configSectionKey)
			if err != nil {
				return fmt.Errorf("load config section ID from key: %w", err)
			}

			err = config.Cloud.DeleteConfigSection(config.Ctx, configSectionID)
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			cmd.Println("Deleted")
			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")

	return cmd
}
