package configsection

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/completer"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
)

func NewCmdCreateConfigSection(cfg *config.Config) *cobra.Command {
	var kind string
	var name string
	var propsSlice []string

	cmd := &cobra.Command{
		Use:   "config_section", // child of `create`
		Short: "Create config section",
		Long:  "Create a snipet of a reutilizable config section that you can attach later to pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			props := propsFromSlice(propsSlice)
			props = append(cloudtypes.Pairs{
				{Key: "name", Value: name},
			}, props...)

			ctx := cmd.Context()
			created, err := cfg.Cloud.CreateConfigSection(ctx, cfg.ProjectID, cloudtypes.CreateConfigSection{
				Kind:       cloudtypes.ConfigSectionKind(kind),
				Properties: props,
			})
			if err != nil {
				return fmt.Errorf("cloud: %w", err)
			}

			fs := cmd.Flags()
			outputFormat := formatters.OutputFormatFromFlags(fs)
			if fn, ok := formatters.ShouldApplyTemplating(outputFormat); ok {
				return fn(cmd.OutOrStdout(), formatters.TemplateFromFlags(fs), created)
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
	formatters.BindFormatFlags(cmd)

	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("name")

	_ = cmd.RegisterFlagCompletionFunc("kind", completer.CompletePluginKinds)
	_ = cmd.RegisterFlagCompletionFunc("name", completer.CompletePluginNames)
	_ = cmd.RegisterFlagCompletionFunc("prop", cfg.Completer.CompletePluginProps)

	return cmd
}

// reSpacesOrEqualSignMoreThanOnce is used to split config section props.
// Example:
//
//	foo=bar -> "foo", "bar"
//	foo bar -> "foo", "bar"
var reSpacesOrEqualSignMoreThanOnce = regexp.MustCompile(`[\s|=]+`)

func propsFromSlice(ss []string) cloudtypes.Pairs {
	if len(ss) == 0 {
		return nil
	}

	var out cloudtypes.Pairs
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
			out = cloudtypes.Pairs{}
		}
		out = append(out, cloudtypes.Pair{
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
