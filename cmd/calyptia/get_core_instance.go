package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/pkg/formatters"
)

func newCmdGetCoreInstances(config *config) *cobra.Command {
	var last uint
	var showIDs bool
	var showMetadata bool
	var environment string
	var outputFormat, goTemplate string

	cmd := &cobra.Command{
		Use:     "core_instances",
		Aliases: []string{"instances", "core_instances"},
		Short:   "Display latest core instances from a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			var params cloud.CoreInstancesParams

			params.Last = &last
			if environmentID != "" {
				params.EnvironmentID = &environmentID
			}

			aa, err := config.cloud.CoreInstances(config.ctx, config.projectID, params)
			if err != nil {
				return fmt.Errorf("could not fetch your core instances: %w", err)
			}

			if strings.HasPrefix(outputFormat, "go-template") {
				return applyGoTemplate(cmd.OutOrStdout(), outputFormat, goTemplate, aa.Items)
			}

			switch outputFormat {
			case "table":
				tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 1, ' ', 0)
				if showIDs {
					fmt.Fprint(tw, "ID\t")
				}
				fmt.Fprint(tw, "NAME\tVERSION\tENVIRONMENT\tPIPELINES\tTAGS\tSTATUS\tAGE")
				if showMetadata {
					fmt.Fprintln(tw, "\tMETADATA")
				} else {
					fmt.Fprintln(tw, "")
				}
				for _, a := range aa.Items {
					if showIDs {
						fmt.Fprintf(tw, "%s\t", a.ID)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\t%s", a.Name, a.Version, a.EnvironmentName, a.PipelinesCount, strings.Join(a.Tags, ","), a.Status, fmtTime(a.CreatedAt))
					if showMetadata {
						metadata, err := filterOutEmptyMetadata(a.Metadata)
						if err != nil {
							continue
						}
						fmt.Fprintf(tw, "\t%s\n", string(metadata))
					} else {
						fmt.Fprintln(tw, "")
					}
				}
				tw.Flush()
			case "json":
				return json.NewEncoder(cmd.OutOrStdout()).Encode(aa.Items)
			case "yml", "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(aa.Items)
			default:
				return fmt.Errorf("unknown output format %q", outputFormat)
			}
			return nil
		},
	}

	fs := cmd.Flags()
	fs.UintVarP(&last, "last", "l", 0, "Last `N` core instances. 0 means no limit")
	fs.BoolVar(&showIDs, "show-ids", false, "Include core instance IDs in table output")
	fs.BoolVar(&showMetadata, "show-metadata", false, "Include core instance metadata in table output")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name.")
	fs.StringVarP(&outputFormat, "output-format", "o", "table", "Output format. Allowed: table, json, yaml, go-template, go-template-file")
	fs.StringVar(&goTemplate, "template", "", "Template string or path to use when -o=go-template, -o=go-template-file. The template format is golang templates\n[http://golang.org/pkg/text/template/#pkg-overview]")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("output-format", formatters.CompleteOutputFormat)

	return cmd
}

func (config *config) completeCoreInstances(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.CoreInstances(config.ctx, config.projectID, cloud.CoreInstancesParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return coreInstanceKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

// coreInstanceKeys returns unique aggregator names first and then IDs.
func coreInstanceKeys(aa []cloud.CoreInstance) []string {
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

func (config *config) loadCoreInstanceID(key string, environmentID string) (string, error) {
	params := cloud.CoreInstancesParams{
		Name: &key,
		Last: ptr(uint(2)),
	}

	if environmentID != "" {
		params.EnvironmentID = &environmentID
	}

	aa, err := config.cloud.CoreInstances(config.ctx, config.projectID, params)
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !validUUID(key) {
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
