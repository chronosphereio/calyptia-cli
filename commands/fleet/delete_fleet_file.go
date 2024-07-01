package fleet

import (
	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
)

func NewCmdDeleteFleetFile(cocfgfig *config.Config) *cobra.Command {
	var confirmed bool
	var fleetKey string
	var name string

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Delete a single file from a fleet by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if !confirmed {
				cmd.Printf("Are you sure you want to delete %q? (y/N) ", fleetKey)
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			fleetID, err := cocfgfig.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := cocfgfig.Cloud.FleetFiles(ctx, fleetID, cloudtypes.FleetFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return cocfgfig.Cloud.DeleteFleetFile(ctx, f.ID)
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.StringVar(&name, "name", "", "File name you want to delete")

	_ = cmd.RegisterFlagCompletionFunc("fleet", cocfgfig.Completer.CompleteFleets)
	_ = cmd.MarkFlagRequired("name")

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.

	return cmd
}
