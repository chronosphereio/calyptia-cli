package pipeline

import (
	"github.com/spf13/cobra"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/config"
)

func NewCmdUpdatePipelineClusterObject(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var clusterObjectKey string
	var environment string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_object",
		Short: "Update a cluster object from a pipeline by its name or ID.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}

			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			clusterObjectID, err := cfg.Completer.LoadClusterObjectID(ctx, clusterObjectKey, environmentID)
			if err != nil {
				return err
			}

			err = cfg.Cloud.UpdatePipelineClusterObjects(ctx, pipelineID, cloudtypes.UpdatePipelineClusterObjects{
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

	_ = cmd.RegisterFlagCompletionFunc("pipeline", cfg.Completer.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("cluster-object", cfg.Completer.CompleteClusterObjects)
	_ = cmd.MarkFlagRequired("cluster-object")
	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}
