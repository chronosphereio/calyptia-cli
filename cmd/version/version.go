package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	DefaultCloudURLStr = "https://cloud-api.calyptia.com"
	Version            = "dev" // To be injected at build time:  -ldflags="-X 'github.com/calyptia/cli/cmd/version.Version=xxx'"
)

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "version",
		Short:         "v",
		SilenceErrors: true,
		SilenceUsage:  true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Version", Version)
		},
	}

	return cmd
}
