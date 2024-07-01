package coreinstance

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

func NewCmdGetCoreInstanceSecrets(cfg *config.Config) *cobra.Command {

	var instanceKey string

	cmd := &cobra.Command{
		Use:   "core_instance_secrets", // get
		Short: "List core instance secrets",
		Long:  "List secrets from a core instance with backward pagination",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			instanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, instanceKey)
			if err != nil {
				return err
			}

			in := cloudtypes.ListCoreInstanceSecrets{
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

			out, err := cfg.Cloud.CoreInstanceSecrets(ctx, in)
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
				return renderCoreInstanceSecrets(cmd.OutOrStdout(), instanceID, in.Before != nil, out)
			}
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&instanceKey, "core-instance", "", "Core instance ID or name")
	fs.UintP("last", "l", 0, "Retrieve the last N secrets")
	fs.String("before", "", "Retrieve secrets before the given cursor")
	formatters.BindFormatFlags(cmd)

	_ = cmd.RegisterFlagCompletionFunc("core-instance", cfg.Completer.CompleteCoreInstances)

	_ = cmd.MarkFlagRequired("core-instance")

	return cmd
}

func renderCoreInstanceSecrets(w io.Writer, coreInstanceID string, paging bool, data cloudtypes.CoreInstanceSecrets) error {
	if len(data.Items) == 0 {
		if paging {
			fmt.Fprintln(w, "End reached.")
			return nil
		}

		fmt.Fprintln(w, "No core instance secrets found.")
		return nil
	}

	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tKEY\tAGE")
	for _, secret := range data.Items {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", secret.ID, secret.Key, formatters.FmtTime(secret.CreatedAt))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "Count: %d\n", data.Count)
	if data.EndCursor != nil {
		fmt.Fprintf(w, "Next page:\n\tcalyptia get core_instance_secrets --core-instance %s --before %s\n", coreInstanceID, *data.EndCursor)
	}

	return nil
}
