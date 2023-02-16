package main

import (
	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
)

func newCmdUpdatePipelineClusterObject(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var clusterObjectKey string
	var environment string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_object",
		Short: "Update a cluster object from a pipeline by its name or ID.",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			pipelineID, err := config.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			clusterObjectID, err := config.LoadClusterObjectID(clusterObjectKey, environmentID)
			if err != nil {
				return err
			}

			err = config.Cloud.UpdatePipelineClusterObjects(config.Ctx, pipelineID, cloud.UpdatePipelineClusterObjects{
				ClusterObjectsIDs: []string{clusterObjectID},
			})
			if err != nil {
				return err
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&pipelineKey, "pipeline", "", "Parent pipeline ID or name")
	fs.StringVar(&clusterObjectKey, "cluster-object", "", "The cluster object ID or Name")
	fs.BoolVar(&encrypt, "encrypt", false, "Encrypt file contents")

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("cluster-object", config.CompleteClusterObjects)
	_ = cmd.MarkFlagRequired("cluster-object")
	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}
