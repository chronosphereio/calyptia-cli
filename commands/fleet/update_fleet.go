package fleet

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdUpdateFleet(cfg *config.Config) *cobra.Command {
	var in cloudtypes.UpdateFleet
	var configFile, configFormat string

	cmd := &cobra.Command{
		Use:               "fleet",
		Short:             "Update fleet by name",
		Long:              "Update a fleet's shared configuration.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteFleets,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetKey := args[0]
			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}
			in.ID = fleetID

			rawConfig, err := readConfig(configFile)
			if err != nil {
				return err
			}
			in.RawConfig = &rawConfig
			format := getFormat(configFile, configFormat)
			in.ConfigFormat = &format

			updated, err := cfg.Cloud.UpdateFleet(ctx, in)
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), updated)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(updated)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(updated)
			default:
				return formatters.RenderUpdated(cmd.OutOrStdout(), updated)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&configFile, "config-file", "fluent-bit.yaml", "Fluent-bit config file")
	fs.StringVar(&configFormat, "config-format", "", "Optional fluent-bit config format (classic, yaml, json)")
	fs.BoolVar(&in.SkipConfigValidation, "skip-config-validation", false, "Option to skip fluent-bit config validation (not recommended)")
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("name")

	_ = cmd.RegisterFlagCompletionFunc("config-format", completeConfigFormat)

	return cmd
}
