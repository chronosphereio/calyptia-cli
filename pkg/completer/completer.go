package completer

import (
	"strings"

	"github.com/calyptia/cli/pkg/helpers"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config"
	"github.com/spf13/cobra"
)

type Completer struct {
}

func (c *Completer) CompletePluginKinds(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"input",
		"filter",
		"output",
	}, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompletePluginNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kind, err := cmd.Flags().GetString("kind")
	if err != nil {
		kind = ""
	}
	return pluginNames(kind), cobra.ShellCompDirectiveNoFileComp
}

func pluginNames(kind string) []string {
	var out []string
	add := func(s string) {
		s = strings.ToLower(s)
		out = append(out, s)
	}
	if kind == "" || kind == "input" {
		for _, in := range fluentbitconfig.DefaultSchema.Inputs {
			add(in.Name)
		}
	}
	if kind == "" || kind == "filter" {
		for _, f := range fluentbitconfig.DefaultSchema.Filters {
			add(f.Name)
		}
	}
	if kind == "" || kind == "output" {
		for _, o := range fluentbitconfig.DefaultSchema.Outputs {
			add(o.Name)
		}
	}

	return helpers.UniqueSlice(out)
}
