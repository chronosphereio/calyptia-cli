package fleet

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdateFleetFile(cfg *config.Config) *cobra.Command {
	var fleetKey string
	var file string

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Update a file from a fleet by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			contents, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			name := filepath.Base(file)

			fleetID, err := cfg.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := cfg.Cloud.FleetFiles(ctx, fleetID, cloudtypes.FleetFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return cfg.Cloud.UpdateFleetFile(ctx, f.ID, cloudtypes.UpdateFleetFile{
						Contents: &contents,
					})
				}
			}

			return fmt.Errorf("fleet file %q not found", name)
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&fleetKey, "fleet", "", "Parent fleet ID or name")
	fs.StringVar(&file, "file", "", "File path. The file you want to update. It must exists already.")

	_ = cmd.RegisterFlagCompletionFunc("fleet", cfg.Completer.CompleteFleets)
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.

	return cmd
}
