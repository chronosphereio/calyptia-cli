package fleet

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdUpdateFleet(config *cfg.Config) *cobra.Command {
	var in types.UpdateFleet
	var configFile, configFormat string
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:               "fleet",
		Short:             "Update fleet by name",
		Long:              "Update a fleet's shared configuration.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteFleets,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			ctx := cmd.Context()

			fleetKey := args[0]
			fleetID, err := completer.LoadFleetID(fleetKey)
			if err != nil {
				return err
			}
			in.ID = fleetID

			cfg, err := readConfig(configFile)
			if err != nil {
				return err
			}
			in.RawConfig = &cfg
			format := getFormat(configFile, configFormat)
			in.ConfigFormat = &format

			updated, err := config.Cloud.UpdateFleet(ctx, in)
			if err != nil {
				return err
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, updated)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				fmt.Fprintln(tw, "ID\tUPDATED-AT")
				fmt.Fprintf(tw, "%s\t%s\n", "0", updated.UpdatedAt.Format(time.RFC822))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(updated)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(updated)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&configFile, "config-file", "fluent-bit.yaml", "Fluent-bit config file")
	fs.StringVar(&configFormat, "config-format", "", "Optional fluent-bit config format (classic, yaml, json)")
	fs.BoolVar(&in.SkipConfigValidation, "skip-config-validation", false, "Option to skip fluent-bit config validation (not recommended)")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("name")

	_ = cmd.RegisterFlagCompletionFunc("config-format", completeConfigFormat)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}
