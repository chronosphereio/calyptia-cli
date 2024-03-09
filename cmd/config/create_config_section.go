package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"

	"github.com/chronosphereio/calyptia-cli/completer"
	cfg "github.com/chronosphereio/calyptia-cli/config"
	"github.com/chronosphereio/calyptia-cli/formatters"
)

func NewCmdCreateConfigSection(config *cfg.Config) *cobra.Command {
	var kind string
	var name string
	var propsSlice []string
	var outputFormat, goTemplate string
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "config_section", // child of `create`
		Short: "Create config section",
		Long:  "Create a snipet of a reutilizable config section that you can attach later to pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			props := propsFromSlice(propsSlice)
			props = append(types.Pairs{
				{Key: "name", Value: name},
			}, props...)

			ctx := cmd.Context()
			created, err := config.Cloud.CreateConfigSection(ctx, config.ProjectID, types.CreateConfigSection{
				Kind:       types.ConfigSectionKind(kind),
				Properties: props,
			})
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return formatters.ApplyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, created)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				return formatters.RenderCreatedTable(cmd.OutOrStdout(), created.ID, created.CreatedAt)
			}
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&kind, "kind", "", "Plugin kind. Either input, filter or output")
	fs.StringVar(&name, "name", "", "Plugin name. See\n[https://docs.fluentbit.io/manual/pipeline]")
	fs.StringSliceVarP(&propsSlice, "prop", "p", nil, "Additional properties; follow the format -p foo=bar -p baz=qux")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("name")

	_ = cmd.RegisterFlagCompletionFunc("kind", completer.CompletePluginKinds)
	_ = cmd.RegisterFlagCompletionFunc("name", completer.CompletePluginNames)
	_ = cmd.RegisterFlagCompletionFunc("prop", completer.CompletePluginProps)

	return cmd
}

// reSpacesOrEqualSignMoreThanOnce is used to split config section props.
// Example:
//
//	foo=bar -> "foo", "bar"
//	foo bar -> "foo", "bar"
var reSpacesOrEqualSignMoreThanOnce = regexp.MustCompile(`[\s|=]+`)

func propsFromSlice(ss []string) types.Pairs {
	if len(ss) == 0 {
		return nil
	}

	var out types.Pairs
	for _, s := range ss {
		ss := reSpacesOrEqualSignMoreThanOnce.Split(s, 2)
		if len(ss) == 0 {
			continue
		}

		key := ss[0]
		var value any

		if len(ss) == 2 {
			value = anyFromString(ss[1])
		}

		if out == nil {
			out = types.Pairs{}
		}
		out = append(out, types.Pair{
			Key:   key,
			Value: value,
		})
	}

	return out
}

func anyFromString(s string) any {
	if strings.EqualFold(s, "true") {
		return true
	}
	if strings.EqualFold(s, "false") {
		return false
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if u, err := strconv.ParseUint(s, 10, 64); err == nil {
		return u
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return s
}
