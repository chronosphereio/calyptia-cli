package configsection

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdUpdateConfigSection(cfg *config.Config) *cobra.Command {
	var propsSlice []string

	cmd := &cobra.Command{
		Use:               "config_section CONFIG_SECTION", // child of `update`
		Short:             "Update a config section",
		Long:              "Update a config section either by the plugin kind:name or by its ID",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteConfigSections,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			configSectionKey := args[0]
			configSectionID, err := cfg.Completer.LoadConfigSectionID(ctx, configSectionKey)
			if err != nil {
				return fmt.Errorf("load config section ID from key: %w", err)
			}

			cs, err := cfg.Cloud.ConfigSection(ctx, configSectionID)
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			props := propsFromSlice(propsSlice)
			props = append(cloudtypes.Pairs{
				{Key: "name", Value: formatters.PairsName(cs.Properties)},
			}, props...)

			updated, err := cfg.Cloud.UpdateConfigSection(ctx, configSectionID, cloudtypes.UpdateConfigSection{
				Properties: &props,
			})
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
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
				return formatters.RenderUpdatedTable(cmd.OutOrStdout(), updated.UpdatedAt)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringSliceVarP(&propsSlice, "prop", "p", nil, "Additional properties; follow the format -p foo=bar -p baz=qux")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("prop", cfg.Completer.CompletePluginProps)

	return cmd
}
