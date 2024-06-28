package coreinstance

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateCoreInstanceSecret(cfg *config.Config) *cobra.Command {
	var instanceKey string
	var key string
	var value string

	cmd := &cobra.Command{
		Use:   "core_instance_secret", // create
		Short: "Create core instance secrets",
		Long:  "Create a secret within a core instance",
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

			instanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, instanceKey)
			if err != nil {
				return err
			}

			out, err := cfg.Cloud.CreateCoreInstanceSecret(ctx, cloudtypes.CreateCoreInstanceSecret{
				CoreInstanceID: instanceID,
				Key:            key,
				Value:          []byte(value),
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
				return formatters.RenderCreated(cmd.OutOrStdout(), out)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&instanceKey, "core-instance", "", "Core instance ID or name")
	fs.StringVar(&key, "key", "", "Secret key")
	fs.StringVar(&value, "value", "", "Secret value")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance")
	_ = cmd.MarkFlagRequired("key")

	return cmd
}

func readPassword() (string, error) {
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}

	return string(b), nil
}
