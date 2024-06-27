package fleet

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdDeleteFleetFile(config *cfg.Config) *cobra.Command {
	var confirmed bool
	var fleetKey string
	var name string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Delete a single file from a fleet by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmed {
				cmd.Printf("Are you sure you want to delete %q? (y/N) ", fleetKey)
				var answer string
				_, err := fmt.Scanln(&answer)
				if err != nil && err.Error() == "unexpected newline" {
					err = nil
				}

				if err != nil {
					return fmt.Errorf("could not to read answer: %v", err)
				}

				answer = strings.TrimSpace(strings.ToLower(answer))
				if answer != "y" && answer != "yes" {
					return nil
				}
			}

			fleetID, err := completer.LoadFleetID(fleetKey)
			if err != nil {
				return err
			}

			ff, err := config.Cloud.FleetFiles(config.Ctx, fleetID, cloud.FleetFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return config.Cloud.DeleteFleetFile(config.Ctx, f.ID)
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", false, "Confirm deletion")
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.StringVar(&name, "name", "", "File name you want to delete")

	_ = cmd.RegisterFlagCompletionFunc("fleet", completer.CompleteFleets)
	_ = cmd.MarkFlagRequired("name")

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.

	return cmd
}
