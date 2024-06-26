package coreinstance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateCoreInstanceFile(cfg *config.Config) *cobra.Command {
	var instanceKey string
	var file string
	var encrypted bool

	cmd := &cobra.Command{
		Use:   "core_instance_file", // create
		Short: "Create core instance files",
		Long:  "Create a file within a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := filepath.Base(file)
			name = strings.TrimSuffix(name, filepath.Ext(name))
			contents, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			instanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, instanceKey)
			if err != nil {
				return err
			}

			fs := cmd.Flags()

			out, err := cfg.Cloud.CreateCoreInstanceFile(ctx, cloudtypes.CreateCoreInstanceFile{
				CoreInstanceID: instanceID,
				Name:           name,
				Contents:       contents,
				Encrypted:      encrypted,
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
	fs.StringVar(&file, "file", "", "File path. You will be able to reference the file from a fluentbit config using its base name without the extension. Ex: `some_dir/my_file.txt` will be referenced as `{{files.my_file}}`")
	fs.BoolVar(&encrypted, "encrypted", false, "Encrypt the file contents")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}
