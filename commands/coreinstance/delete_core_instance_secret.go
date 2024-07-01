package coreinstance

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/formatters"
)

func NewCmdDeleteCoreInstanceSecret(cfg *config.Config) *cobra.Command {
	var confirmed bool
	var instanceKey string
	var key string

	cmd := &cobra.Command{
		Use:   "core_instance_secret", // delete
		Short: "Delete core instance secret",
		Long:  "Delete a secret within a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if !confirmed {
				cmd.Printf("Are you sure you want to delete secret %q? (y/N) ", key)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			instanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, instanceKey)
			if err != nil {
				return err
			}

			secrets, err := cfg.Cloud.CoreInstanceSecrets(ctx, cloudtypes.ListCoreInstanceSecrets{
				CoreInstanceID: instanceID,
			})
			if err != nil {
				return err
			}

			var secretID string
			for _, s := range secrets.Items {
				if s.Key == key {
					secretID = s.ID
					break
				}
			}

			if secretID == "" {
				cmd.Println("Secret not found.")
				return nil
			}

			out, err := cfg.Cloud.DeleteCoreInstanceSecret(ctx, secretID)
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), out)
			}

			switch outputFormat {
			case formatters.OutputFormatJSON:
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			case formatters.OutputFormatYAML:
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
			default:
				return formatters.RenderDeleted(cmd.OutOrStdout(), out)
			}
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&instanceKey, "core-instance", "", "Parent core instance ID or name")
	fs.StringVar(&key, "key", "", "Secret key")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}
