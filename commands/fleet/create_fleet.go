package fleet

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateFleet(cfg *config.Config) *cobra.Command {
	var in cloudtypes.CreateFleet
	var configFile, configFormat string
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:   "fleet",
		Short: "Create fleet",
		Long:  "Create a new fleet with a shared fluent-bit config in where agents can be attached and share that config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var err error
			in.RawConfig, err = readConfig(configFile)
			if err != nil {
				return err
			}

			in.ConfigFormat = getFormat(configFile, configFormat)
			in.ProjectID = cfg.ProjectID

			created, err := cfg.Cloud.CreateFleet(ctx, in)
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, created)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tCREATED-AT")
				fmt.Fprintf(tw, "%s\t%s\n", created.ID, created.CreatedAt.Format(time.RFC822))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&in.Name, "name", "", "Name")
	fs.StringVar(&in.MinFluentBitVersion, "min-fluent-bit-version", "", "Optional minimum fluent-bit version that agents must satisfy to join this fleet")
	fs.StringVar(&configFile, "config-file", "fluent-bit.yaml", "Fluent-bit config file")
	fs.StringVar(&configFormat, "config-format", "", "Optional fluent-bit config format (classic, yaml, json)")
	fs.StringSliceVar(&in.Tags, "tags", nil, "Optional tags for this fleet")
	fs.BoolVar(&in.SkipConfigValidation, "skip-config-validation", false, "Option to skip fluent-bit config validation (not recommended)")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("name")

	_ = cmd.RegisterFlagCompletionFunc("config-format", completeConfigFormat)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

func readConfig(filename string) (string, error) {
	out, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func completeConfigFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"yaml", "json", "classic"}, cobra.ShellCompDirectiveNoFileComp
}

func getFormat(configFile, configFormat string) cloudtypes.ConfigFormat {
	if configFormat == "" || strings.EqualFold(configFormat, "auto") {
		return cloudtypes.ConfigFormat(strings.TrimPrefix(filepath.Ext(configFile), "."))
	}
	return cloudtypes.ConfigFormat(configFormat)
}
