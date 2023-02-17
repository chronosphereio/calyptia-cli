package completer

import (
	"fmt"
	"strings"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/config"
	"github.com/calyptia/cli/pkg/formatters"
	"github.com/calyptia/cli/pkg/helpers"
	"github.com/calyptia/core-images-index/go-index"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config"
	"github.com/spf13/cobra"
)

type Completer struct {
	Config *config.Config
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

func (c *Completer) CompletePluginProps(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var kind, name string

	if len(args) == 1 {
		ctx := cmd.Context()
		key := args[0]
		id, err := c.Config.LoadConfigSectionID(ctx, key)
		if err != nil {
			cobra.CompError(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		cs, err := c.Config.Cloud.ConfigSection(ctx, id)
		if err != nil {
			cobra.CompError(fmt.Sprintf("cloud: %v", err))
			return nil, cobra.ShellCompDirectiveError
		}

		kind = string(cs.Kind)
		name = helpers.PairsName(cs.Properties)
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

	return helpers.PluginProps(kind, name), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteConfigSections(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	cc, err := c.Config.Cloud.ConfigSections(ctx, c.Config.ProjectID, types.ConfigSectionsParams{})
	if err != nil {
		cobra.CompErrorln(fmt.Sprintf("cloud: %v", err))
		return nil, cobra.ShellCompDirectiveError
	}

	if len(cc.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return configSectionKeys(cc.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := c.Config.Cloud.Environments(c.Config.Ctx, c.Config.ProjectID, types.EnvironmentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return environmentNames(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteFleets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ff, err := c.Config.Cloud.Fleets(c.Config.Ctx, types.FleetsParams{
		ProjectID: c.Config.ProjectID,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(ff.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return fleetKeys(ff.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteCoreInstances(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := c.Config.Cloud.CoreInstances(c.Config.Ctx, c.Config.ProjectID, types.CoreInstancesParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return CoreInstanceKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteResourceProfiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// TODO: complete resource profiles.
	return []string{
		types.ResourceProfileHighPerformanceGuaranteedDelivery,
		types.ResourceProfileHighPerformanceOptimalThroughput,
		types.ResourceProfileBestEffortLowResource,
	}, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteCoreContainerVersion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	containerIndex, err := index.NewContainer()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	vv, err := containerIndex.All(c.Config.Ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return vv, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompletePipelines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := c.Config.FetchAllPipelines()
	if err != nil {
		cobra.CompError(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return PipelinesKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

// pipelinesKeys returns unique pipeline names first and then IDs.
func PipelinesKeys(aa []types.Pipeline) []string {
	namesCount := map[string]int{}
	for _, a := range aa {
		if _, ok := namesCount[a.Name]; ok {
			namesCount[a.Name] += 1
			continue
		}

		namesCount[a.Name] = 1
	}

	var out []string

	for _, a := range aa {
		var nameIsUnique bool
		for name, count := range namesCount {
			if a.Name == name && count == 1 {
				nameIsUnique = true
				break
			}
		}
		if nameIsUnique {
			out = append(out, a.Name)
			continue
		}

		out = append(out, a.ID)
	}

	return out
}

func fleetKeys(ff []types.Fleet) []string {
	var out []string
	for _, f := range ff {
		out = append(out, f.Name)
	}
	return out
}

// environmentNames returns unique environment names that belongs to a project.
func environmentNames(aa []types.Environment) []string {
	var out []string
	for _, a := range aa {
		out = append(out, a.Name)
	}
	return out
}

func configSectionKeys(cc []types.ConfigSection) []string {
	kindNameCounts := map[string]uint{}
	for _, cs := range cc {
		kindName := formatters.ConfigSectionKindName(cs)
		if _, ok := kindNameCounts[kindName]; ok {
			kindNameCounts[kindName]++
			continue
		}

		kindNameCounts[kindName] = 1
	}

	var out []string
	for _, cs := range cc {
		kindName := formatters.ConfigSectionKindName(cs)
		if count, ok := kindNameCounts[kindName]; ok && count == 1 {
			out = append(out, kindName)
		} else {
			out = append(out, cs.ID)
		}
	}

	return out
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

// coreInstanceKeys returns unique aggregator names first and then IDs.
func CoreInstanceKeys(aa []types.CoreInstance) []string {
	namesCount := map[string]int{}
	for _, a := range aa {
		if _, ok := namesCount[a.Name]; ok {
			namesCount[a.Name] += 1
			continue
		}

		namesCount[a.Name] = 1
	}

	var out []string

	for _, a := range aa {
		var nameIsUnique bool
		for name, count := range namesCount {
			if a.Name == name && count == 1 {
				nameIsUnique = true
				break
			}
		}
		if nameIsUnique {
			out = append(out, a.Name)
			continue
		}

		out = append(out, a.ID)
	}

	return out
}
