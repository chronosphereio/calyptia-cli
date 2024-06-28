package version

import (
	"github.com/spf13/cobra"
)

var (
	DefaultCloudURLStr = "https://cloud-api.calyptia.com"
	Version            = "dev" // To be injected at build time:  -ldflags="-X 'github.com/calyptia/cli/commands/version.Version=xxx'"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "version",
		Short:        "Returns currenty Calyptia CLI version.",
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(Version)
		},
	}

	return cmd
}
