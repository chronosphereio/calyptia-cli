package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
)

func (config *config) fetchAllClusterObjects() ([]cloud.ClusterObject, error) {
	aa, err := config.cloud.CoreInstances(config.ctx, config.projectID, cloud.CoreInstancesParams{})
	if err != nil {
		return nil, fmt.Errorf("could not prefetch core-instances: %w", err)
	}

	if len(aa.Items) == 0 {
		return nil, nil
	}

	var clusterobjects []cloud.ClusterObject
	var mu sync.Mutex

	g, gctx := errgroup.WithContext(config.ctx)
	for _, a := range aa.Items {
		a := a
		g.Go(func() error {
			objects, err := config.cloud.ClusterObjects(gctx, a.ID,
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

func (config *config) completeClusterObjects(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pp, err := config.fetchAllClusterObjects()
	if err != nil {
		cobra.CompError(err.Error())
		return nil, cobra.ShellCompDirectiveError
	}

	if pp == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return clusterObjectsKeys(pp), cobra.ShellCompDirectiveNoFileComp
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

func (config *config) loadClusterObjectID(key string, environmentID string) (string, error) {
	aa, err := config.fetchAllClusterObjects()
	if err != nil {
		return "", err
	}

	objs := clusterObjectsUniqueByName(aa)
	for _, obj := range objs {
		if obj.Name == key {
			return obj.ID, nil
		}
	}

	return "", fmt.Errorf("unable to find unique key")
}

func newCmdGetClusterObjects(config *config) *cobra.Command {
	var coreInstanceKey string
	var last uint
	var outputFormat, goTemplate string
	var environment string
	var showIDs bool

	cmd := &cobra.Command{
		Use:   "cluster_objects",
		Short: "Get cluster objects",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			coreInstanceID, err := config.loadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			co, err := config.cloud.ClusterObjects(config.ctx, coreInstanceID, cloud.ClusterObjectParams{
				Last: &last,
			})
			if err != nil {
				return fmt.Errorf("could not fetch your cluster objects: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, co.Items)
			}

			switch outputFormat {
			case "table":
				{
					tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
					if showIDs {
						fmt.Fprintf(tw, "ID\t")
					}
					fmt.Fprintln(tw, "NAME\tKIND\tCREATED AT")
					for _, c := range co.Items {
						if showIDs {
							fmt.Fprintf(tw, "%s\t", c.ID)
						}
						fmt.Fprintf(tw, "%s\t%s\t%s\n", c.Name, string(c.Kind), fmtTime(c.CreatedAt))
					}
					tw.Flush()
				}
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(co.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(co.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceKey, "core-instance", "", "Core Instance to list cluster objects from")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` cluster objects. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include status IDs in table output")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.MarkFlagRequired("core-instance")

	return cmd
}
