package main

import (
	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdUpdatePipelineClusterObjects(config *config) *cobra.Command {
	var pipelineKey string
	var clusterObjectKey string
	var environment string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_objects",
		Short: "Update a file from a pipeline by its name",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			pipelineID, err := config.loadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			clusterObjectID, err := config.loadClusterObjectID(clusterObjectKey, environmentID)
			if err != nil {
				return err
			}

			err = config.cloud.UpdatePipelineClusterObjects(config.ctx, pipelineID, cloud.UpdatePipelineClusterObjects{
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

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.completePipelines)
	_ = cmd.RegisterFlagCompletionFunc("cluster-object", config.completeClusterObjects)
	_ = cmd.MarkFlagRequired("cluster-object")
	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}
