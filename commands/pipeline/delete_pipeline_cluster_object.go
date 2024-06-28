package pipeline

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/config"
)

func NewCmdDeletePipelineClusterObject(cfg *config.Config) *cobra.Command {
	var pipelineKey string
	var clusterObjectKey string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_object",
		Short: "Delete pipeline cluster object",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			pipelineID, err := cfg.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			clusterObjectID, err := cfg.Completer.LoadClusterObjectID(ctx, clusterObjectKey)
			if err != nil {
				return err
			}

			err = cfg.Cloud.DeletePipelineClusterObjects(ctx, pipelineID, clusterObjectID)
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
