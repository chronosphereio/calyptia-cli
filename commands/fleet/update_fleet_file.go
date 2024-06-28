package fleet

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdUpdateFleetFile(config *cfg.Config) *cobra.Command {
	var fleetKey string
	var file string

	cmd := &cobra.Command{
		Use:   "fleet_file",
		Short: "Update a file from a fleet by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			contents, err := cfg.ReadFile(file)
			if err != nil {
				return err
			}

			name := filepath.Base(file)

			fleetID, err := config.Completer.LoadFleetID(ctx, fleetKey)
			if err != nil {
				return err
			}

			ff, err := config.Cloud.FleetFiles(ctx, fleetID, cloud.FleetFilesParams{})
			if err != nil {
				return err
			}

			for _, f := range ff.Items {
				if f.Name == name {
					return config.Cloud.UpdateFleetFile(ctx, f.ID, cloud.UpdateFleetFile{
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

	_ = cmd.RegisterFlagCompletionFunc("fleet", config.Completer.CompleteFleets)
	_ = cmd.MarkFlagRequired("file")

	_ = cmd.MarkFlagRequired("fleet") // TODO: use default fleet key from config cmd.

	return cmd
}
