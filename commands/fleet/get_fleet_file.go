package fleet

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetFleetFiles(config *cfg.Config) *cobra.Command {
	var fleetKey string
	var last uint
	var outputFormat, goTemplate string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "fleet_files",
		Short: "Get fleet files",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetID, err := config.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := config.Cloud.FleetFiles(ctx, fleetID, cloud.FleetFilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your fleet files: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, ff.Items)
			}

			switch outputFormat {
			case "table":
				renderFleetFiles(cmd.OutOrStdout(), ff.Items, showIDs)
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` fleet files. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)
	_ = cmd.RegisterFlagCompletionFunc("fleet", config.Completer.CompleteFleets)

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.

	return cmd
}

func NewCmdGetFleetFile(config *cfg.Config) *cobra.Command {
	var fleetKey string
	var name string
	var outputFormat, goTemplate string
	var showIDs, onlyContents bool

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Get a single file from a fleet by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetID, err := config.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := config.Cloud.FleetFiles(ctx, fleetID, cloud.FleetFilesParams{})
			if err != nil {
				return fmt.Errorf("could not find your fleet file by name: %w", err)
			}

			var file cloud.FleetFile
			var found bool
			for _, f := range ff.Items {
				if f.Name == name {
					file = f
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("could not find fleet file %q", name)
			}

			if onlyContents {
				cmd.Print(string(file.Contents))
				return nil
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, file)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tAGE")
				if showIDs {
					fmt.Fprintf(tw, "%s\t", file.ID)
				}
				fmt.Fprintf(tw, "%s\t%s\n", file.Name, formatters.FmtTime(file.CreatedAt))
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(file)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(file)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.StringVar(&name, "name", "", "File name")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.BoolVar(&onlyContents, "only-contents", false, "Only print file contents")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("fleet", config.Completer.CompleteFleets)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func renderFleetFiles(w io.Writer, ff []cloud.FleetFile, showIDs bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		fmt.Fprint(tw, "ID\t")
	}
	fmt.Fprintln(tw, "NAME\tAGE")
	for _, f := range ff {
		if showIDs {
			fmt.Fprintf(tw, "%s\t", f.ID)
		}
		fmt.Fprintf(tw, "%s\t%s\n", f.Name, formatters.FmtTime(f.CreatedAt))
	}
	tw.Flush()
}
