package main

import (
	"github.com/calyptia/cloud"
	"github.com/spf13/cobra"
)

func (config *config) completeResourceProfiles(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// TODO: complete resource profiles.
	return []string{
		string(cloud.HighPerformanceGuaranteedDelivery),
		string(cloud.HighPerformanceOptimalThroughput),
		string(cloud.BestEffortLowResource),
	}, cobra.ShellCompDirectiveNoFileComp
}
