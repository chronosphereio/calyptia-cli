package completer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/formatters"
	"github.com/calyptia/cli/helpers"
	"github.com/calyptia/cli/slice"
	"github.com/calyptia/core-images-index/go-index"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config"
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
		id, err := c.LoadConfigSectionID(ctx, key)
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

func (c *Completer) CompleteMembers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	mm, err := c.Config.Cloud.Members(ctx, c.Config.ProjectID, types.MembersParams{})
	if err != nil {
		cmd.PrintErrf("fetch members: %v\n", err)
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

func (c *Completer) CompletePermissions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"create:*", "read:*", "update:*", "delete:*"}, cobra.ShellCompDirectiveNoFileComp
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
	pp, err := c.FetchAllPipelines()
	if err != nil {
		cobra.CompError(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return PipelinesKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteClusterObjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := c.FetchAllClusterObjects()
	if err != nil {
		cobra.CompError(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return clusterObjectsKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteTraceSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ss, err := c.fetchAllTraceSessions()
	if err != nil {
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

func (c *Completer) LoadCoreInstanceID(key string, environmentID string) (string, error) {
	params := types.CoreInstancesParams{
		Name: &key,
		Last: config.Ptr(uint(2)),
	}

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := c.Config.Cloud.CoreInstances(c.Config.Ctx, c.Config.ProjectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !config.ValidUUID(key) {
		if len(aa.Items) != 0 {
			return "", fmt.Errorf("ambiguous core instance name %q, use ID instead", key)
		}

		return "", fmt.Errorf("could not find core instance %q", key)
	}

	if len(aa.Items) == 1 {
		return aa.Items[0].ID, nil
	}

	return key, nil
}

func (c *Completer) fetchAllTraceSessions() ([]types.TraceSession, error) {
	pp, err := c.FetchAllPipelines()
	if err != nil {
		return nil, err
	}

	if len(pp) == 0 {
		return nil, nil
	}

	var ss []types.TraceSession
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(c.Config.Ctx)
	for _, pip := range pp {
		a := pip
		g.Go(func() error {
			got, err := c.Config.Cloud.TraceSessions(gctx, a.ID, types.TraceSessionsParams{})
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

func (c *Completer) CompletePipelinePlugins(pipelineKey string, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pipelineID, err := c.LoadPipelineID(pipelineKey)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	pipeline, err := c.Config.Cloud.Pipeline(c.Config.Ctx, pipelineID, types.PipelineParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	conf, err := fluentbitconfig.ParseAs(pipeline.Config.RawConfig, fluentbitconfig.Format(pipeline.Config.ConfigFormat))
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// TODO: use instance id instead of name.

	var out []string
	for _, byName := range conf.Pipeline.Inputs {
		for name := range byName {
			out = append(out, name)
		}
	}

	for _, byName := range conf.Pipeline.Filters {
		for name := range byName {
			out = append(out, name)
		}
	}

	for _, byName := range conf.Pipeline.Outputs {
		for name := range byName {
			out = append(out, name)
		}
	}

	return out, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteAgents(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := c.Config.Cloud.Agents(c.Config.Ctx, c.Config.ProjectID, types.AgentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return AgentsKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteSecretIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pipelines, err := c.FetchAllPipelines()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var secrets []types.PipelineSecret
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(c.Config.Ctx)
	for _, pip := range pipelines {
		pip := pip
		g.Go(func() error {
			ss, err := c.Config.Cloud.PipelineSecrets(gctx, pip.ID, types.PipelineSecretsParams{})
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
	cc, err := c.Config.Cloud.ConfigSections(ctx, c.Config.ProjectID, types.ConfigSectionsParams{})
	if err != nil {
		return "", fmt.Errorf("cloud: %w", err)
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

func (c *Completer) LoadEnvironmentID(environmentName string) (string, error) {
	aa, err := c.Config.Cloud.Environments(c.Config.Ctx, c.Config.ProjectID, types.EnvironmentsParams{
		Name: &environmentName,
		Last: config.Ptr(uint(1)),
	})
	if err != nil {
		return "", err
	}

	if len(aa.Items) == 0 {
		return "", fmt.Errorf("could not find environment %q", environmentName)

	}

	return aa.Items[0].ID, nil
}

func (c *Completer) LoadPipelineID(pipelineKey string) (string, error) {
	pp, err := c.Config.Cloud.ProjectPipelines(c.Config.Ctx, c.Config.ProjectID, types.PipelinesParams{
		Name: &pipelineKey,
		Last: config.Ptr(uint(2)),
	})
	if err != nil {
		return "", err
	}

	if len(pp.Items) != 1 && !config.ValidUUID(pipelineKey) {
		if len(pp.Items) != 0 {
			return "", fmt.Errorf("ambiguous pipeline name %q, use ID instead", pipelineKey)
		}

		return "", fmt.Errorf("could not find pipeline %q", pipelineKey)
	}

	if len(pp.Items) == 1 {
		return pp.Items[0].ID, nil
	}

	return pipelineKey, nil
}

func (c *Completer) FetchAllPipelines() ([]types.Pipeline, error) {
	aa, err := c.Config.Cloud.CoreInstances(c.Config.Ctx, c.Config.ProjectID, types.CoreInstancesParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch core-instances: %w", err)
	}

	if len(aa.Items) == 0 {
		return nil, nil
	}

	var pipelines []types.Pipeline
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(c.Config.Ctx)
	for _, a := range aa.Items {
		a := a
		g.Go(func() error {
			got, err := c.Config.Cloud.Pipelines(gctx, a.ID, types.PipelinesParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			pipelines = append(pipelines, got.Items...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var uniquePipelines []types.Pipeline
	pipelineIDs := map[string]struct{}{}
	for _, pip := range pipelines {
		if _, ok := pipelineIDs[pip.ID]; !ok {
			uniquePipelines = append(uniquePipelines, pip)
			pipelineIDs[pip.ID] = struct{}{}
		}
	}

	return uniquePipelines, nil
}

func (c *Completer) LoadFleetID(key string) (string, error) {
	ff, err := c.Config.Cloud.Fleets(c.Config.Ctx, types.FleetsParams{
		ProjectID: c.Config.ProjectID,
		Name:      &key,
		Last:      config.Ptr(uint(1)),
	})
	if err != nil {
		return "", err
	}

	if len(ff.Items) == 1 {
		return ff.Items[0].ID, nil
	}

	if !config.ValidUUID(key) {
		return "", fmt.Errorf("could not find fleet %q", key)
	}

	return key, nil
}

func (c *Completer) LoadAgentID(agentKey string, environmentID string) (string, error) {
	var err error

	var params types.AgentsParams

	params.Last = config.Ptr(uint(2))
	params.Name = &agentKey

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := c.Config.Cloud.Agents(c.Config.Ctx, c.Config.ProjectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !config.ValidUUID(agentKey) {
		if len(aa.Items) != 0 {
			return "", fmt.Errorf("ambiguous agent name %q, use ID instead", agentKey)
		}
		return "", fmt.Errorf("could not find agent %q", agentKey)
	}

	if len(aa.Items) == 1 {
		return aa.Items[0].ID, nil
	}

	return agentKey, nil
}

func (c *Completer) LoadClusterObjectID(key string, environmentID string) (string, error) {
	aa, err := c.FetchAllClusterObjects()
	if err != nil {
		return "", err
	}

	objs := clusterObjectsUniqueByName(aa)

	if config.ValidUUID(key) {
		for _, obj := range objs {
			if obj.ID == key {
				return obj.ID, nil
			}
		}
	}

	for _, obj := range objs {
		if obj.Name == key {
			return obj.ID, nil
		}
	}

	return "", fmt.Errorf("unable to find unique key")
}

func (c *Completer) FetchAllClusterObjects() ([]types.ClusterObject, error) {
	aa, err := c.Config.Cloud.CoreInstances(c.Config.Ctx, c.Config.ProjectID, types.CoreInstancesParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch core-instances: %w", err)
	}

	if len(aa.Items) == 0 {
		return nil, nil
	}

	var clusterobjects []types.ClusterObject
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(c.Config.Ctx)
	for _, a := range aa.Items {
		a := a
		g.Go(func() error {
			objects, err := c.Config.Cloud.ClusterObjects(gctx, a.ID,
				types.ClusterObjectParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			clusterobjects = append(clusterobjects, objects.Items...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	var uniqueClusterObjects []types.ClusterObject
	clusterObjectIDs := map[string]struct{}{}
	for _, coid := range clusterobjects {
		if _, ok := clusterObjectIDs[coid.ID]; !ok {
			uniqueClusterObjects = append(uniqueClusterObjects, coid)
			clusterObjectIDs[coid.ID] = struct{}{}
		}
	}

	return uniqueClusterObjects, nil
}

// ClusterObjectsUnique returns unique cluster object names.
func clusterObjectsUniqueByName(aa []types.ClusterObject) []types.ClusterObject {
	namesCount := map[string]int{}
	for _, a := range aa {
		if _, ok := namesCount[a.Name]; !ok {
			namesCount[a.Name] = 0
		}
		namesCount[a.Name]++
	}

	var out []types.ClusterObject
	for _, a := range aa {
		for name, count := range namesCount {
			if a.Name == name && count == 1 {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

// agentsKeys returns unique agent names first and then IDs.
func AgentsKeys(aa []types.Agent) []string {
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

// ClusterObjectsKeys returns unique cluster object names first and then IDs.
func clusterObjectsKeys(aa []types.ClusterObject) []string {
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

	return slice.Unique(out)
}

// CoreInstanceKeys returns unique aggregator names first and then IDs.
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
