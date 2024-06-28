package coreinstance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/formatters"
)

func NewCmdDeleteCoreInstanceFile(cfg *config.Config) *cobra.Command {

	var confirmed bool
	var instanceKey string
	var name string

	cmd := &cobra.Command{
		Use:   "core_instance_file", // delete
		Short: "Delete core instance file",
		Long:  "Delete a file within a core instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := filepath.Base(name)
			name = strings.TrimSuffix(name, filepath.Ext(name))

			if !confirmed {
				cmd.Printf("Are you sure you want to delete file %q? (y/N) ", name)
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

			files, err := cfg.Cloud.CoreInstanceFiles(ctx, cloudtypes.ListCoreInstanceFiles{
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
				cmd.Println("File not found.")
				return nil
			}

			out, err := cfg.Cloud.DeleteCoreInstanceFile(ctx, fileID)
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
	fs.StringVar(&name, "name", "", "Name of the file to delete")

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance")
	_ = cmd.MarkFlagRequired("name")

	return cmd
}
