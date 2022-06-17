package main

import (
	"fmt"

	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func (config *config) completeEnvironments(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.Environments(config.ctx, config.projectID, cloud.EnvironmentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return environmentsKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

// environmentsKeys returns unique environment names first and then IDs.
func environmentsKeys(aa []cloud.Environment) []string {
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

func (config *config) loadEnvironmentID(environmentKey string) (string, error) {
	aa, err := config.cloud.Environments(config.ctx, config.projectID, cloud.EnvironmentsParams{
		Name: &environmentKey,
		Last: ptr(uint64(2)),
	})
	if err != nil {
		return "", err
	}

	if len(aa.Items) != 1 && !validUUID(environmentKey) {
		if len(aa.Items) != 0 {
			return "", fmt.Errorf("ambiguous core instance name %q, use ID instead", environmentKey)
		}

		return "", fmt.Errorf("could not find core instance %q", environmentKey)
	}

	if len(aa.Items) == 1 {
		return aa.Items[0].ID, nil
	}

	return environmentKey, nil
}
