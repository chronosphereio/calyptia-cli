package formatters

import (
	"fmt"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/helpers"
	"github.com/spf13/cobra"
)

func CompleteOutputFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml", "go-template"}, cobra.ShellCompDirectiveNoFileComp
}

func ConfigSectionKindName(cs types.ConfigSection) string {
	return fmt.Sprintf("%s:%s", cs.Kind, helpers.PairsName(cs.Properties))
}
