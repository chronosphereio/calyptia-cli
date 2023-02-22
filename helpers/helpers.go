package helpers

import (
	"fmt"
	"strings"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/slice"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config"
	"golang.org/x/exp/slices"
)

// pluginProps -
// TODO: exclude already defined property.
func PluginProps(kind, name string) []string {
	if kind == "" || name == "" {
		return nil
	}

	var out []string
	add := func(sec fluentbitconfig.SchemaSection) {
		if !strings.EqualFold(sec.Name, name) {
			return
		}

		for _, p := range sec.Properties.Options {
			out = append(out, p.Name)
		}
		for _, p := range sec.Properties.Networking {
			out = append(out, p.Name)
		}
		for _, p := range sec.Properties.NetworkTLS {
			out = append(out, p.Name)
		}
	}
	switch kind {
	case "input":
		for _, in := range fluentbitconfig.DefaultSchema.Inputs {
			add(in)
		}
	case "filter":
		for _, f := range fluentbitconfig.DefaultSchema.Filters {
			add(f)
		}
	case "output":
		for _, o := range fluentbitconfig.DefaultSchema.Outputs {
			add(o)
		}
	}

	// common properties that are not in the schema.
	out = append(out, "Alias")
	if kind == "input" {
		out = append(out, "Tag")
	} else if kind == "filter" || kind == "output" {
		out = append(out, "Match", "Match_Regex")
	}

	slices.Sort(out)
	slices.Compact(out)

	return slice.UniqueSlice(out)
}

func PairsName(pp types.Pairs) string {
	if v, ok := pp.Get("Name"); ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
