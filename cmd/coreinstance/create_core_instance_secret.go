package coreinstance

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"

	"github.com/chronosphereio/calyptia-cli/completer"
	cfg "github.com/chronosphereio/calyptia-cli/config"
	"github.com/chronosphereio/calyptia-cli/formatters"
)

func NewCmdCreateCoreInstanceSecret(config *cfg.Config) *cobra.Command {
	loader := completer.Completer{Config: config}

	var instanceKey string
	var key string
	var value string

	cmd := &cobra.Command{
		Use:   "core_instance_secret", // create
		Short: "Create core instance secrets",
		Long:  "Create a secret within a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			instanceID, err := loader.LoadCoreInstanceID(instanceKey, "")
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			out, err := config.Cloud.CreateCoreInstanceSecret(ctx, types.CreateCoreInstanceSecret{
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

	_ = cmd.RegisterFlagCompletionFunc("core-instance", loader.CompleteCoreInstances)

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
