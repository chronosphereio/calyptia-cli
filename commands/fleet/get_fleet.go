package fleet

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdGetFleets(cfg *config.Config) *cobra.Command {
	var name, before string
	var tags []string
	var last uint
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "fleets", // calyptia get fleets
		Short: "Fleets",
		Long:  "List all the fleets from the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var in cloudtypes.FleetsParams

			fs := cmd.Flags()
			if fs.Changed("name") {
				in.Name = &name
			}
			if fs.Changed("tags") {
				in.SetTags(tags)
			}
			if fs.Changed("last") {
				in.Last = &last
			}
			if fs.Changed("before") {
				in.Before = &before
			}

			in.ProjectID = cfg.ProjectID

			fleets, err := cfg.Cloud.Fleets(ctx, in)
			if err != nil {
				return err
			}

			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), fleets)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(fleets)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(fleets)
			default:
				return renderFleetsTable(cmd.OutOrStdout(), fleets, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&name, "name", "", "Filter fleets by name")
	fs.StringSliceVar(&tags, "tags", nil, "Filter fleets by tags")
	fs.UintVar(&last, "last", 0, "Paginate and retrieve only the last N fleets")
	fs.StringVar(&before, "before", "", "Paginate and retrieve the fleets before the given cursor")
	fs.BoolVar(&showIDs, "show-ids", false, "Show fleets IDs. Only applies when output format is table")
	formatters.BindFormatFlags(cmd)

	return cmd
}

func NewCmdGetFleet(cfg *config.Config) *cobra.Command {
	var showIDs bool

	cmd := &cobra.Command{
		Use:               "fleet FLEET", // calyptia get fleets
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cfg.Completer.CompleteFleets,
		Short:             "Display a Fleet Fleet",
		Long:              "Display a Fleet by ID or name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			fleetKey := args[0]
			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			fleet, err := cfg.Cloud.Fleet(ctx, cloudtypes.FleetParams{FleetID: fleetID})
			if err != nil {
				return err
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), fleet)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(fleet)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(fleet)
			default:
				return renderFleetsTable(cmd.OutOrStdout(),
					cloudtypes.Fleets{Items: []cloudtypes.Fleet{fleet}}, showIDs)
			}
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&showIDs, "show-ids", false, "Show fleets IDs. Only applies when output format is table")
	formatters.BindFormatFlags(cmd)

	return cmd
}

func renderFleetsTable(w io.Writer, fleets cloudtypes.Fleets, showIDs bool) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	if showIDs {
		if _, err := fmt.Fprint(tw, "ID\t"); err != nil {
			return err
		}
	}
	fmt.Fprintln(tw, "NAME\tTAGS\tAGE")
	for _, fleet := range fleets.Items {
		if showIDs {
			_, err := fmt.Fprintf(tw, "%s\t", fleet.ID)
			if err != nil {
				return err
			}
		}
		_, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", fleet.Name, strings.Join(fleet.Tags, ", "), formatters.FmtTime(fleet.CreatedAt))
		if err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if fleets.EndCursor != nil {
		_, err := fmt.Fprintf(w, "\n\n# Previous page:\n\tcalyptia get fleets --before %s\n", *fleets.EndCursor)
		if err != nil {
			return err
		}
	}

	return nil
}
