package pipeline

import (
	"github.com/spf13/cobra"

	cfg "github.com/calyptia/cli/config"
)

func NewCmdDeletePipelineClusterObject(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var clusterObjectKey string
	var environment string
	var encrypt bool

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_object",
		Short: "Delete pipeline cluster object",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}

			pipelineID, err := config.Completer.LoadPipelineID(ctx, pipelineKey)
			if err != nil {
				return err
			}

			clusterObjectID, err := config.Completer.LoadClusterObjectID(ctx, clusterObjectKey, environmentID)
			if err != nil {
				return err
			}

			err = config.Cloud.DeletePipelineClusterObjects(ctx, pipelineID, clusterObjectID)
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

	_ = cmd.RegisterFlagCompletionFunc("pipeline", config.Completer.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("cluster-object", config.Completer.CompleteClusterObjects)
	_ = cmd.MarkFlagRequired("cluster-object")
	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}