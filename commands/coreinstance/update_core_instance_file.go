package coreinstance

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdUpdateCoreInstanceFile(config *cfg.Config) *cobra.Command {
	loader := completer.Completer{Config: config}

	var instanceKey string
	var file string

	cmd := &cobra.Command{
		Use:   "core_instance_file", // update
		Short: "Update core instance file",
		Long:  "Update a file within a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := filepath.Base(file)
			name = strings.TrimSuffix(name, filepath.Ext(name))
			contents, err := cfg.ReadFile(file)
			if err != nil {
				return err
			}

			instanceID, err := loader.LoadCoreInstanceID(instanceKey, "")
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			files, err := config.Cloud.CoreInstanceFiles(ctx, types.ListCoreInstanceFiles{
				CoreInstanceID: instanceID,
			})
			if err != nil {
				return err
			}

			var fileID string
			for _, f := range files.Items {
				if f.Name == name {
					fileID = f.ID
					break
				}
			}

			if fileID == "" {
				return errors.New("file not found")
			}

			fs := cmd.Flags()
			var encrypted bool
			if fs.Changed("encrypted") {
				encrypted, err = fs.GetBool("encrypted")
				if err != nil {
					return err
				}
			}

			out, err := config.Cloud.UpdateCoreInstanceFile(ctx, types.UpdateCoreInstanceFile{
				ID:        fileID,
				Contents:  &contents,
				Encrypted: &encrypted,
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
	fs.StringVar(&file, "file", "", "File path. The file you want to update. It must exists already.")
	fs.Bool("encrypted", false, "Encrypt file contents")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", loader.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("file")

	return cmd
}
