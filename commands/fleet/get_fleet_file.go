package fleet

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetFleetFiles(cfg *config.Config) *cobra.Command {
	var fleetKey string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "fleet_files",
		Short: "Get fleet files",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.FleetFiles(ctx, fleetID, cloudtypes.FleetFilesParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your fleet files: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), ff)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(ff.Items)
			default:
				return renderFleetFiles(cmd.OutOrStdout(), ff.Items, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` fleet files. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("fleet", cfg.Completer.CompleteFleets)

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.

	return cmd
}

func NewCmdGetFleetFile(cfg *config.Config) *cobra.Command {
	var fleetKey string
	var name string
	var showIDs, onlyContents bool

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Get a single file from a fleet by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.FleetFiles(ctx, fleetID, cloudtypes.FleetFilesParams{})
			if err != nil {
				return fmt.Errorf("could not find your fleet file by name: %w", err)
			}

			var file cloudtypes.FleetFile
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

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), file)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(file)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(file)
			default:
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprintln(tw, "NAME\tAGE")
				if showIDs {
					fmt.Fprintf(tw, "%s\t", file.ID)
				}
				fmt.Fprintf(tw, "%s\t%s\n", file.Name, formatters.FmtTime(file.CreatedAt))
				return tw.Flush()
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.StringVar(&name, "name", "", "File name")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.BoolVar(&onlyContents, "only-contents", false, "Only print file contents")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("fleet", cfg.Completer.CompleteFleets)

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.
	_ = cmd.MarkFlagRequired("name")

	return cmd
}

func renderFleetFiles(w io.Writer, ff []cloudtypes.FleetFile, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	fmt.Fprintln(tw, "NAME\tAGE")
	for _, f := range ff {
		if showIDs {
			if _, err := fmt.Fprintf(tw, "%s\t", f.ID); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%s\n", f.Name, formatters.FmtTime(f.CreatedAt))
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}
