package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v2"

	"github.com/calyptia/api/types"
	fluentbit_config "github.com/calyptia/go-fluentbit-config"
)

func newCmdCreateConfigSection(config *config) *cobra.Command {
	var kind string
	var name string
	var propsSlice []string
	var outputFormat, goTemplate string

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
			created, err := config.cloud.CreateConfigSection(ctx, config.projectID, types.CreateConfigSection{
				Kind:       types.ConfigSectionKind(kind),
				Properties: props,
			})
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, created)
			}

			switch outputFormat {
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(created)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(created)
			default:
				return renderCreatedTable(cmd.OutOrStdout(), created.ID, created.CreatedAt)
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

	_ = cmd.RegisterFlagCompletionFunc("kind", completePluginKinds)
	_ = cmd.RegisterFlagCompletionFunc("name", completePluginNames)
	_ = cmd.RegisterFlagCompletionFunc("prop", config.completePluginProps)

	return cmd
}

func completePluginKinds(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"input",
		"filter",
		"output",
	}, cobra.ShellCompDirectiveNoFileComp
}

func (config *config) completePluginProps(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var kind, name string

	if len(args) == 1 {
		ctx := cmd.Context()
		key := args[0]
		id, err := config.loadConfigSectionID(ctx, key)
		if err != nil {
			cobra.CompError(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		cs, err := config.cloud.ConfigSection(ctx, id)
		if err != nil {
			cobra.CompError(fmt.Sprintf("cloud: %v", err))
			return nil, cobra.ShellCompDirectiveError
		}

		kind = string(cs.Kind)
		name = pairsName(cs.Properties)
	} else {
		var err error
		kind, err = cmd.Flags().GetString("kind")
		if err != nil {
			kind = ""
		}

		name, err = cmd.Flags().GetString("name")
		if err != nil {
			name = ""
		}
	}

	return pluginProps(kind, name), cobra.ShellCompDirectiveNoFileComp
}

func completePluginNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
		for _, in := range fluentbit_config.DefaultSchema.Inputs {
			add(in.Name)
		}
	}
	if kind == "" || kind == "filter" {
		for _, f := range fluentbit_config.DefaultSchema.Filters {
			add(f.Name)
		}
	}
	if kind == "" || kind == "output" {
		for _, o := range fluentbit_config.DefaultSchema.Outputs {
			add(o.Name)
		}
	}

	return uniqueSlice(out)
}

// pluginProps -
// TODO: exclude already defined property.
func pluginProps(kind, name string) []string {
	if kind == "" || name == "" {
		return nil
	}

	var out []string
	add := func(sec fluentbit_config.SchemaSection) {
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
		for _, in := range fluentbit_config.DefaultSchema.Inputs {
			add(in)
		}
	case "filter":
		for _, f := range fluentbit_config.DefaultSchema.Filters {
			add(f)
		}
	case "output":
		for _, o := range fluentbit_config.DefaultSchema.Outputs {
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

	return uniqueSlice(out)
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
