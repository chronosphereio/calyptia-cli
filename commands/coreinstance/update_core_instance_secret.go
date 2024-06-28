package coreinstance

import (
	"encoding/json"
	"errors"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdUpdateCoreInstanceSecret(config *cfg.Config) *cobra.Command {

	var instanceKey string
	var key, value string

	cmd := &cobra.Command{
		Use:   "core_instance_secret", // update
		Short: "Update core instance secret",
		Long:  "Update a secret within a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fs := cmd.Flags()
			if !fs.Changed("value") {
				cmd.Print("Enter secret value: ")
				var err error
				if value, err = readPassword(); err != nil {
					cmd.Println()
					return err
				}

				cmd.Println()
			}

			instanceID, err := config.Completer.LoadCoreInstanceID(ctx, instanceKey, "")
			if err != nil {
				return err
			}

			secrets, err := config.Cloud.CoreInstanceSecrets(ctx, types.ListCoreInstanceSecrets{
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
				return errors.New("secret not found")
			}

			newValue := []byte(value)
			out, err := config.Cloud.UpdateCoreInstanceSecret(ctx, types.UpdateCoreInstanceSecret{
				ID:    secretID,
				Value: &newValue,
			})
			if err != nil {
				return err
			}

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
				return formatters.RenderUpdated(cmd.OutOrStdout(), out)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&instanceKey, "core-instance", "", "Parent core instance ID or name")
	fs.StringVar(&key, "key", "", "Secret key")
	fs.StringVar(&value, "value", "", "Secret value")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", config.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("key")

	return cmd
}
