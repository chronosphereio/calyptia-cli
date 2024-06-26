package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateFleetFile(cfg *config.Config) *cobra.Command {
	var fleetKey string
	var file string

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Create a new file within a fleet",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			name := filepath.Base(file)

			contents, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			out, err := cfg.Cloud.CreateFleetFile(ctx, fleetID, cloudtypes.CreateFleetFile{
				Name:     name,
				Contents: contents,
			})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), out)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tAGE")
				fmt.Fprintf(tw, "%s\t%s\n", out.ID, formatters.FmtTime(out.CreatedAt))
				tw.Flush()

				return nil
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&fleetKey, "fleet", "", "Fleet ID or name")
	fs.StringVar(&file, "file", "", "File path. You will be able to reference the file from a fluentbit config using its base name without the extension. Ex: `some_dir/my_file.txt` will be referenced as `{{files.my_file}}`")
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("fleet")
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.RegisterFlagCompletionFunc("fleet", cfg.Completer.CompleteFleets)

	return cmd
}
