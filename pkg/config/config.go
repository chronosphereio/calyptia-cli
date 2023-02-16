package config

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/calyptia/api/client"
	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/core-images-index/go-index"
	fluentbitconfig "github.com/calyptia/go-fluentbit-config"
	"github.com/hako/durafmt"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Ctx          context.Context
	BaseURL      string
	Cloud        *client.Client
	ProjectToken string
	ProjectID    string
}

func (config *Config) CompletePluginProps(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var kind, name string

	if len(args) == 1 {
		ctx := cmd.Context()
		key := args[0]
		id, err := config.LoadConfigSectionID(ctx, key)
		if err != nil {
			cobra.CompError(err.Error())
			return nil, cobra.ShellCompDirectiveError
		}

		cs, err := config.Cloud.ConfigSection(ctx, id)
		if err != nil {
			cobra.CompError(fmt.Sprintf("cloud: %v", err))
			return nil, cobra.ShellCompDirectiveError
		}

		kind = string(cs.Kind)
		name = PairsName(cs.Properties)
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

func (config *Config) LoadConfigSectionID(ctx context.Context, key string) (string, error) {
	cc, err := config.Cloud.ConfigSections(ctx, config.ProjectID, cloud.ConfigSectionsParams{})
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
		kindName := configSectionKindName(cs)
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

func (config *Config) CompleteConfigSections(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ctx := cmd.Context()
	cc, err := config.Cloud.ConfigSections(ctx, config.ProjectID, cloud.ConfigSectionsParams{})
	if err != nil {
		cobra.CompErrorln(fmt.Sprintf("cloud: %v", err))
		return nil, cobra.ShellCompDirectiveError
	}

	if len(cc.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return configSectionKeys(cc.Items), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) CompleteEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.Cloud.Environments(config.Ctx, config.ProjectID, cloud.EnvironmentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return environmentNames(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) CompleteSecretIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pipelines, err := config.FetchAllPipelines()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var secrets []cloud.PipelineSecret
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.Ctx)
	for _, pip := range pipelines {
		pip := pip
		g.Go(func() error {
			ss, err := config.Cloud.PipelineSecrets(gctx, pip.ID, cloud.PipelineSecretsParams{})
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

// environmentNames returns unique environment names that belongs to a project.
func environmentNames(aa []cloud.Environment) []string {
	var out []string
	for _, a := range aa {
		out = append(out, a.Name)
	}
	return out
}

func (config *Config) LoadEnvironmentID(environmentName string) (string, error) {
	aa, err := config.Cloud.Environments(config.Ctx, config.ProjectID, cloud.EnvironmentsParams{
		Name: &environmentName,
		Last: Ptr(uint(1)),
	})
	if err != nil {
		return "", err
	}

	if len(aa.Items) == 0 {
		return "", fmt.Errorf("could not find environment %q", environmentName)

	}

	return aa.Items[0].ID, nil
}

func (config *Config) CompleteCoreContainerVersion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	containerIndex, err := index.NewContainer()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	vv, err := containerIndex.All(config.Ctx)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return vv, cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) LoadCoreInstanceID(key string, environmentID string) (string, error) {
	params := cloud.CoreInstancesParams{
		Name: &key,
		Last: Ptr(uint(2)),
	}

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := config.Cloud.CoreInstances(config.Ctx, config.ProjectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !ValidUUID(key) {
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

func (config *Config) LoadPipelineID(pipelineKey string) (string, error) {
	pp, err := config.Cloud.ProjectPipelines(config.Ctx, config.ProjectID, cloud.PipelinesParams{
		Name: &pipelineKey,
		Last: Ptr(uint(2)),
	})
	if err != nil {
		return "", err
	}

	if len(pp.Items) != 1 && !ValidUUID(pipelineKey) {
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

func (config *Config) CompletePipelines(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.FetchAllPipelines()
	if err != nil {
		cobra.CompError(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return PipelinesKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) FetchAllPipelines() ([]cloud.Pipeline, error) {
	aa, err := config.Cloud.CoreInstances(config.Ctx, config.ProjectID, cloud.CoreInstancesParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch core-instances: %w", err)
	}

	if len(aa.Items) == 0 {
		return nil, nil
	}

	var pipelines []cloud.Pipeline
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.Ctx)
	for _, a := range aa.Items {
		a := a
		g.Go(func() error {
			got, err := config.Cloud.Pipelines(gctx, a.ID, cloud.PipelinesParams{})
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

	var uniquePipelines []cloud.Pipeline
	pipelineIDs := map[string]struct{}{}
	for _, pip := range pipelines {
		if _, ok := pipelineIDs[pip.ID]; !ok {
			uniquePipelines = append(uniquePipelines, pip)
			pipelineIDs[pip.ID] = struct{}{}
		}
	}

	return uniquePipelines, nil
}

func (config *Config) CompleteCoreInstances(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.Cloud.CoreInstances(config.Ctx, config.ProjectID, cloud.CoreInstancesParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return CoreInstanceKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) CompleteResourceProfiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// TODO: complete resource profiles.
	return []string{
		cloud.ResourceProfileHighPerformanceGuaranteedDelivery,
		cloud.ResourceProfileHighPerformanceOptimalThroughput,
		cloud.ResourceProfileBestEffortLowResource,
	}, cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) CompletePipelinePlugins(pipelineKey string, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pipelineID, err := config.LoadPipelineID(pipelineKey)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	pipeline, err := config.Cloud.Pipeline(config.Ctx, pipelineID, cloud.PipelineParams{})
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

func (config *Config) CompleteAgents(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.Cloud.Agents(config.Ctx, config.ProjectID, cloud.AgentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return AgentsKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) LoadFleetID(key string) (string, error) {
	ff, err := config.Cloud.Fleets(config.Ctx, cloud.FleetsParams{
		ProjectID: config.ProjectID,
		Name:      &key,
		Last:      Ptr(uint(1)),
	})
	if err != nil {
		return "", err
	}

	if len(ff.Items) == 1 {
		return ff.Items[0].ID, nil
	}

	if !ValidUUID(key) {
		return "", fmt.Errorf("could not find fleet %q", key)
	}

	return key, nil
}

// agentsKeys returns unique agent names first and then IDs.
func AgentsKeys(aa []cloud.Agent) []string {
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

func (config *Config) LoadAgentID(agentKey string, environmentID string) (string, error) {
	var err error

	var params cloud.AgentsParams

	params.Last = Ptr(uint(2))
	params.Name = &agentKey

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := config.Cloud.Agents(config.Ctx, config.ProjectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !ValidUUID(agentKey) {
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

func (config *Config) LoadClusterObjectID(key string, environmentID string) (string, error) {
	aa, err := config.FetchAllClusterObjects()
	if err != nil {
		return "", err
	}

	objs := clusterObjectsUniqueByName(aa)

	if ValidUUID(key) {
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

func (config *Config) FetchAllClusterObjects() ([]cloud.ClusterObject, error) {
	aa, err := config.Cloud.CoreInstances(config.Ctx, config.ProjectID, cloud.CoreInstancesParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch core-instances: %w", err)
	}

	if len(aa.Items) == 0 {
		return nil, nil
	}

	var clusterobjects []cloud.ClusterObject
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.Ctx)
	for _, a := range aa.Items {
		a := a
		g.Go(func() error {
			objects, err := config.Cloud.ClusterObjects(gctx, a.ID,
				cloud.ClusterObjectParams{})
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

	var uniqueClusterObjects []cloud.ClusterObject
	clusterObjectIDs := map[string]struct{}{}
	for _, coid := range clusterobjects {
		if _, ok := clusterObjectIDs[coid.ID]; !ok {
			uniqueClusterObjects = append(uniqueClusterObjects, coid)
			clusterObjectIDs[coid.ID] = struct{}{}
		}
	}

	return uniqueClusterObjects, nil
}

func (config *Config) CompleteClusterObjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.FetchAllClusterObjects()
	if err != nil {
		cobra.CompError(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return clusterObjectsKeys(pp), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) CompleteFleets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ff, err := config.Cloud.Fleets(config.Ctx, cloud.FleetsParams{
		ProjectID: config.ProjectID,
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(ff.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return fleetKeys(ff.Items), cobra.ShellCompDirectiveNoFileComp
}

func (config *Config) CompleteTraceSessions(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	ss, err := config.fetchAllTraceSessions()
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

func (config *Config) fetchAllTraceSessions() ([]cloud.TraceSession, error) {
	pp, err := config.FetchAllPipelines()
	if err != nil {
		return nil, err
	}

	if len(pp) == 0 {
		return nil, nil
	}

	var ss []cloud.TraceSession
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.Ctx)
	for _, pip := range pp {
		a := pip
		g.Go(func() error {
			got, err := config.Cloud.TraceSessions(gctx, a.ID, cloud.TraceSessionsParams{})
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

func fleetKeys(ff []cloud.Fleet) []string {
	var out []string
	for _, f := range ff {
		out = append(out, f.Name)
	}
	return out
}

// ClusterObjectsKeys returns unique cluster object names first and then IDs.
func clusterObjectsKeys(aa []cloud.ClusterObject) []string {
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

// ClusterObjectsUnique returns unique cluster object names.
func clusterObjectsUniqueByName(aa []cloud.ClusterObject) []cloud.ClusterObject {
	namesCount := map[string]int{}
	for _, a := range aa {
		if _, ok := namesCount[a.Name]; !ok {
			namesCount[a.Name] = 0
		}
		namesCount[a.Name]++
	}

	var out []cloud.ClusterObject
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

func AgentStatus(lastMetricsAddedAt *time.Time, start time.Duration) string {
	var status string
	if lastMetricsAddedAt == nil || lastMetricsAddedAt.IsZero() {
		status = "inactive"
	} else if lastMetricsAddedAt.Before(time.Now().Add(start)) {
		status = fmt.Sprintf("inactive for %s", durafmt.ParseShort(time.Since(*lastMetricsAddedAt)).LimitFirstN(1))
	} else {
		status = "active"
	}
	return status
}

// coreInstanceKeys returns unique aggregator names first and then IDs.
func CoreInstanceKeys(aa []cloud.CoreInstance) []string {
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
func PipelinesKeys(aa []cloud.Pipeline) []string {
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

func configSectionKeys(cc []cloud.ConfigSection) []string {
	kindNameCounts := map[string]uint{}
	for _, cs := range cc {
		kindName := configSectionKindName(cs)
		if _, ok := kindNameCounts[kindName]; ok {
			kindNameCounts[kindName]++
			continue
		}

		kindNameCounts[kindName] = 1
	}

	var out []string
	for _, cs := range cc {
		kindName := configSectionKindName(cs)
		if count, ok := kindNameCounts[kindName]; ok && count == 1 {
			out = append(out, kindName)
		} else {
			out = append(out, cs.ID)
		}
	}

	return out
}

func configSectionKindName(cs cloud.ConfigSection) string {
	return fmt.Sprintf("%s:%s", cs.Kind, PairsName(cs.Properties))
}

func PairsName(pp cloud.Pairs) string {
	if v, ok := pp.Get("Name"); ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
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
	slices.Compact(out)

	return UniqueSlice(out)
}

func UniqueSlice[S ~[]E, E comparable](s S) S {
	m := map[E]struct{}{}

	var out S
	for _, item := range s {
		if _, ok := m[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}

func Env(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func CompleteOutputFormat(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"table", "json", "yaml", "go-template"}, cobra.ShellCompDirectiveNoFileComp
}

func ReadFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}

	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("could not read contents: %w", err)
	}

	return b, nil
}

var reUUID4 = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

func Ptr[T any](p T) *T { return &p }

func ValidUUID(s string) bool {
	return reUUID4.MatchString(s)
}
