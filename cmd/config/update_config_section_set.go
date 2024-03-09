package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/chronosphereio/calyptia-cli/completer"
	cfg "github.com/chronosphereio/calyptia-cli/config"
)

func NewCmdUpdateConfigSectionSet(config *cfg.Config) *cobra.Command {
	var configSectionKeys []string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "config_section_set PIPELINE", // child of `update`
		Short:             "Update a config section set",
		Long:              "Attaches a list of config sections to a pipeline",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompletePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineKey := args[0]
			pipelineID, err := completer.LoadPipelineID(pipelineKey)
			if err != nil {
				return fmt.Errorf("load pipeline ID from key: %w", err)
			}

			var configSectionIDs []string
			for _, key := range configSectionKeys {
				id, err := completer.LoadConfigSectionID(ctx, key)
				if err != nil {
					return fmt.Errorf("load config section ID from key: %w", err)
				}

				configSectionIDs = append(configSectionIDs, id)
			}

			err = config.Cloud.UpdateConfigSectionSet(ctx, pipelineID, configSectionIDs...)
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			cmd.Println("Updated")
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringSliceVarP(&configSectionKeys, "config-section", "c", nil, "List of config sections.\nFormat is either: -c one -c two, or -c one,two.\nEither the plugin kind:name or the ID")

	_ = cmd.RegisterFlagCompletionFunc("config-section", completer.CompleteConfigSections)

	return cmd
}
