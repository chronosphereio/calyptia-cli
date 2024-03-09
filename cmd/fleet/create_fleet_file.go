package fleet

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"

	"github.com/chronosphereio/calyptia-cli/completer"
	cfg "github.com/chronosphereio/calyptia-cli/config"
	"github.com/chronosphereio/calyptia-cli/formatters"
)

func NewCmdCreateFleetFile(config *cfg.Config) *cobra.Command {
	var fleetKey string
	var file string
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Create a new file within a fleet",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := filepath.Base(file)

			contents, err := cfg.ReadFile(file)
			if err != nil {
				return err
			}

			fleetID, err := completer.LoadFleetID(fleetKey)
			if err != nil {
				return err
			}

			out, err := config.Cloud.CreateFleetFile(config.Ctx, fleetID, cloud.CreateFleetFile{
				Name:     name,
				Contents: contents,
			})
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, out)
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
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("fleet")
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.RegisterFlagCompletionFunc("fleet", completer.CompleteFleets)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
