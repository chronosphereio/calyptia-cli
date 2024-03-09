package coreinstance

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/calyptia/api/types"

	"github.com/chronosphereio/calyptia-cli/completer"
	cfg "github.com/chronosphereio/calyptia-cli/config"
	"github.com/chronosphereio/calyptia-cli/formatters"
)

func NewCmdGetCoreInstanceFiles(config *cfg.Config) *cobra.Command {
	loader := completer.Completer{Config: config}

	var instanceKey string

	cmd := &cobra.Command{
		Use:   "core_instance_files", // get
		Short: "List core instance files",
		Long:  "List files from a core instance with backward pagination",
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceID, err := loader.LoadCoreInstanceID(instanceKey, "")
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			in := types.ListCoreInstanceFiles{
				CoreInstanceID: instanceID,
			}

			fs := cmd.Flags()
			if fs.Changed("last") {
				last, err := fs.GetUint("last")
				if err != nil {
					return err
				}
				in.Last = &last
			}

			if fs.Changed("before") {
				before, err := fs.GetString("before")
				if err != nil {
					return err
				}
				in.Before = &before
			}

			out, err := config.Cloud.CoreInstanceFiles(ctx, in)
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
				return renderCoreInstanceFiles(cmd.OutOrStdout(), instanceID, in.Before != nil, out)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&instanceKey, "core-instance", "", "Core instance ID or name")
	fs.UintP("last", "l", 0, "Retrieve the last N files")
	fs.String("before", "", "Retrieve files before the given cursor")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("core-instance", loader.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance")

	return cmd
}

func renderCoreInstanceFiles(w io.Writer, coreInstanceID string, paging bool, data types.CoreInstanceFiles) error {
	if len(data.Items) == 0 {
		if paging {
			fmt.Fprintln(w, "End reached.")
			return nil
		}

		fmt.Fprintln(w, "No core instance files found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tENCRYPTED\tAGE")
	for _, file := range data.Items {
		fmt.Fprintf(tw, "%s\t%s\t%v\t%s\n", file.ID, file.Name, file.Encrypted, formatters.FmtTime(file.CreatedAt))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Count: %d\n", data.Count)
	if data.EndCursor != nil {
		fmt.Fprintf(w, "Next page:\n\tcalyptia get core_instance_files --core-instance %s --before %s\n", coreInstanceID, *data.EndCursor)
	}

	return nil
}
