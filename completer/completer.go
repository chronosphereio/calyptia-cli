package completer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	cloudclient "github.com/calyptia/api/client"
	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/formatters"
	"github.com/calyptia/cli/pointer"
	"github.com/calyptia/cli/uuid"
	"github.com/calyptia/core-images-index/go-index"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config/v2"
)

type Completer struct {
	Cloud     *cloudclient.Client
	ProjectID string
}

func (c *Completer) complete(resource cloudtypes.SearchResource, cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	results, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  resource,
		Term:      toComplete,
	})
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	return searchResultsNames(results), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteAgents(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return c.complete(cloudtypes.SearchResourceAgent, cmd, args, toComplete)
}

func (c *Completer) CompleteFleets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return c.complete(cloudtypes.SearchResourceFleet, cmd, args, toComplete)
}

func (c *Completer) CompleteCoreInstances(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return c.complete(cloudtypes.SearchResourceCoreInstance, cmd, args, toComplete)
}

func (c *Completer) CompletePipelines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return c.complete(cloudtypes.SearchResourcePipeline, cmd, args, toComplete)
}

func (c *Completer) CompleteClusterObjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return c.complete(cloudtypes.SearchResourceClusterObject, cmd, args, toComplete)
}

func (c *Completer) CompletePluginProps(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var kind, name string

	if len(args) == 1 {
		ctx := cmd.Context()
		key := args[0]
		id, err := c.LoadConfigSectionID(ctx, key)
		if err != nil {
			cobra.CompErrorln(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		cs, err := c.Cloud.ConfigSection(ctx, id)
		if err != nil {
			cobra.CompErrorln(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		kind = string(cs.Kind)
		name = formatters.PairsName(cs.Properties)
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

func (c *Completer) CompleteConfigSections(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	cc, err := c.Cloud.ConfigSections(ctx, c.ProjectID, cloudtypes.ConfigSectionsParams{})
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if len(cc.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return configSectionKeys(cc.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteMembers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	mm, err := c.Cloud.Members(ctx, c.ProjectID, cloudtypes.MembersParams{})
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if len(mm.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	out := make([]string, 0, len(mm.Items))
	for _, m := range mm.Items {
		out = append(out, fmt.Sprintf("%s\t%s", m.ID, m.User.Email))
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	aa, err := c.Cloud.Environments(ctx, c.ProjectID, cloudtypes.EnvironmentsParams{})
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return environmentNames(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteCoreContainerVersion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	containerIndex, err := index.NewContainer()
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	vv, err := containerIndex.All(ctx)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	return vv, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteCoreOperatorVersion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	operatorIndex, err := index.NewOperator()
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	vv, err := operatorIndex.All(ctx)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	return vv, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteTraceSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	ss, err := c.fetchAllTraceSessions(ctx)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if ss == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	out := make([]string, len(ss))
	for i, p := range ss {
		out[i] = p.ID
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) LoadCoreInstanceID(ctx context.Context, key string) (string, error) {
	result, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourceCoreInstance,
		Term:      key,
	})
	if err != nil {
		return "", err
	}

	if len(result) == 0 {
		return "", fmt.Errorf("could not find core instance %q", key)
	}

	if len(result) != 1 && !uuid.Valid(key) {
		return "", fmt.Errorf("ambiguous core instance name %q, use ID instead", key)
	}

	if len(result) == 1 {
		return result[0].ID, nil
	}

	return key, nil
}

func (c *Completer) fetchAllTraceSessions(ctx context.Context) ([]cloudtypes.TraceSession, error) {
	pipelines, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourcePipeline,
	})
	if err != nil {
		return nil, err
	}

	if len(pipelines) == 0 {
		return nil, nil
	}

	var ss []cloudtypes.TraceSession
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(ctx)
	for _, pip := range pipelines {
		a := pip
		g.Go(func() error {
			got, err := c.Cloud.TraceSessions(gctx, a.ID, cloudtypes.TraceSessionsParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			ss = append(ss, got.Items...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return ss, nil
}

func (c *Completer) CompletePipelinePlugins(ctx context.Context, pipelineKey string, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pipelineID, err := c.LoadPipelineID(ctx, pipelineKey)
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	pipeline, err := c.Cloud.Pipeline(ctx, pipelineID, cloudtypes.PipelineParams{})
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	conf, err := fluentbitconfig.ParseAs(pipeline.Config.RawConfig, fluentbitconfig.Format(pipeline.Config.ConfigFormat))
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	// TODO: use instance id instead of name.

	var out []string
	for _, plugin := range conf.Pipeline.Inputs {
		out = append(out, plugin.Name)
	}

	for _, plugin := range conf.Pipeline.Filters {
		out = append(out, plugin.Name)
	}

	for _, plugin := range conf.Pipeline.Outputs {
		out = append(out, plugin.Name)
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteSecretIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	pipelines, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourcePipeline,
	})
	if err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	var secrets []cloudtypes.PipelineSecret
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	for _, pip := range pipelines {
		pip := pip
		g.Go(func() error {
			ss, err := c.Cloud.PipelineSecrets(gctx, pip.ID, cloudtypes.PipelineSecretsParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			secrets = append(secrets, ss.Items...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		cobra.CompErrorln(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	var uniqueSecretsIDs []string
	secretIDs := map[string]struct{}{}
	for _, s := range secrets {
		if _, ok := secretIDs[s.ID]; !ok {
			uniqueSecretsIDs = append(uniqueSecretsIDs, s.ID)
			secretIDs[s.ID] = struct{}{}
		}
	}

	return uniqueSecretsIDs, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) LoadConfigSectionID(ctx context.Context, key string) (string, error) {
	cc, err := c.Cloud.ConfigSections(ctx, c.ProjectID, cloudtypes.ConfigSectionsParams{})
	if err != nil {
		return "", err
	}

	if len(cc.Items) == 0 {
		return "", errors.New("cloud: no config sections yet")
	}

	for _, cs := range cc.Items {
		if key == cs.ID {
			return cs.ID, nil
		}
	}

	var foundID string
	var foundCount uint

	for _, cs := range cc.Items {
		kindName := formatters.ConfigSectionKindName(cs)
		if kindName == key {
			foundID = cs.ID
			foundCount++
		}
	}

	if foundCount > 1 {
		return "", fmt.Errorf("ambiguous config section %q, try using the ID", key)
	}

	if foundCount == 0 {
		return "", fmt.Errorf("could not find config section with key %q", key)
	}

	return foundID, nil
}

func (c *Completer) LoadEnvironmentID(ctx context.Context, environmentName string) (string, error) {
	aa, err := c.Cloud.Environments(ctx, c.ProjectID, cloudtypes.EnvironmentsParams{
		Name: &environmentName,
		Last: pointer.From(uint(1)),
	})
	if err != nil {
		return "", err
	}

	if len(aa.Items) == 0 {
		return "", fmt.Errorf("could not find environment %q", environmentName)
	}

	return aa.Items[0].ID, nil
}

func (c *Completer) LoadPipelineID(ctx context.Context, pipelineKey string) (string, error) {
	results, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourcePipeline,
		Term:      pipelineKey,
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("could not find pipeline %q", pipelineKey)
	}

	if len(results) != 1 && !uuid.Valid(pipelineKey) {
		return "", fmt.Errorf("ambiguous pipeline name %q, use ID instead", pipelineKey)
	}

	if len(results) == 1 {
		return results[0].ID, nil
	}

	return pipelineKey, nil
}

func (c *Completer) LoadFleetID(ctx context.Context, key string) (string, error) {
	results, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourceFleet,
		Term:      key,
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("could not find fleet %q", key)
	}

	if len(results) != 1 && !uuid.Valid(key) {
		return "", fmt.Errorf("ambiguous fleet name %q, use ID instead", key)
	}

	if len(results) == 1 {
		return results[0].ID, nil
	}

	return key, nil
}

func (c *Completer) LoadAgentID(ctx context.Context, agentKey string) (string, error) {
	results, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourceAgent,
		Term:      agentKey,
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("could not find agent %q", agentKey)
	}

	if len(results) != 1 && !uuid.Valid(agentKey) {
		return "", fmt.Errorf("ambiguous agent name %q, use ID instead", agentKey)
	}

	if len(results) == 1 {
		return results[0].ID, nil
	}

	return agentKey, nil
}

func (c *Completer) LoadClusterObjectID(ctx context.Context, key string) (string, error) {
	results, err := c.Cloud.Search(ctx, cloudtypes.SearchQuery{
		ProjectID: c.ProjectID,
		Resource:  cloudtypes.SearchResourceClusterObject,
		Term:      key,
	})
	if err != nil {
		return "", err
	}

	if len(results) == 0 {
		return "", fmt.Errorf("could not find cluster object %q", key)
	}

	if len(results) != 1 && !uuid.Valid(key) {
		return "", fmt.Errorf("ambiguous cluster object name %q, use ID instead", key)
	}

	if len(results) == 1 {
		return results[0].ID, nil
	}

	return key, nil
}

// AgentsKeys returns unique agent names first and then IDs.
func AgentsKeys(aa []cloudtypes.Agent) []string {
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
		count, ok := namesCount[a.Name]
		if !ok {
			continue
		}

		if count == 1 {
			out = append(out, a.Name)
			continue
		}

		out = append(out, a.ID)
	}

	return out
}

// PipelinesKeys returns unique pipeline names first and then IDs.
func PipelinesKeys(aa []cloudtypes.Pipeline) []string {
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

func searchResultsNames(rr []cloudtypes.SearchResult) []string {
	var out []string
	for _, r := range rr {
		out = append(out, r.Name)
	}
	return out
}

// environmentNames returns unique environment names that belongs to a project.
func environmentNames(aa []cloudtypes.Environment) []string {
	var out []string
	for _, a := range aa {
		out = append(out, a.Name)
	}
	return out
}

func configSectionKeys(cc []cloudtypes.ConfigSection) []string {
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

	slices.Sort(out)
	return slices.Compact(out)
}

// CoreInstanceKeys returns unique aggregator names first and then IDs.
func CoreInstanceKeys(aa []cloudtypes.CoreInstance) []string {
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

func CompletePluginKinds(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"input",
		"filter",
		"output",
	}, cobra.ShellCompDirectiveNoFileComp
}

func CompletePluginNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	kind, err := cmd.Flags().GetString("kind")
	if err != nil {
		kind = ""
	}
	return pluginNames(kind), cobra.ShellCompDirectiveNoFileComp
}

func CompleteResourceProfiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		cloudtypes.ResourceProfileHighPerformanceGuaranteedDelivery,
		cloudtypes.ResourceProfileHighPerformanceOptimalThroughput,
		cloudtypes.ResourceProfileBestEffortLowResource,
	}, cobra.ShellCompDirectiveNoFileComp
}

func CompletePermissions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return cloudtypes.AllPermissions(), cobra.ShellCompDirectiveNoFileComp
}

// pluginProps -
// TODO: exclude already defined property.
func pluginProps(kind, name string) []string {
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
	return slices.Compact(out)
}
