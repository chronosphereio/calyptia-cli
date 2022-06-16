package main

import (
	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
)

func (config *config) completeEnvironmentIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	aa, err := config.cloud.Environments(config.ctx, config.projectID, cloud.EnvironmentsParams{})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(aa.Items) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return environmentsKeys(aa.Items), cobra.ShellCompDirectiveNoFileComp
}

// environmentsKeys returns unique aggregator names first and then IDs.
func environmentsKeys(aa []cloud.Environment) []string {
	var out []string

	for _, a := range aa {
		out = append(out, a.ID)
	}

	return out
}
