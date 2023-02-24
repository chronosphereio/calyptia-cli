package pipeline

import (
	"github.com/spf13/cobra"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
)

func NewCmdDeletePipelineClusterObject(config *cfg.Config) *cobra.Command {
	var pipelineKey string
	var clusterObjectKey string
	var environment string
	var encrypt bool
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:   "pipeline_cluster_object",
		Short: "Delete pipeline cluster object",
		RunE: func(cmd *cobra.Command, args []string) error {
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			pipelineID, err := completer.LoadPipelineID(pipelineKey)
			if err != nil {
				return err
			}

			clusterObjectID, err := completer.LoadClusterObjectID(clusterObjectKey, environmentID)
			if err != nil {
				return err
			}

			err = config.Cloud.DeletePipelineClusterObjects(config.Ctx, pipelineID, clusterObjectID)
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

	_ = cmd.RegisterFlagCompletionFunc("pipeline", completer.CompletePipelines)
	_ = cmd.RegisterFlagCompletionFunc("cluster-object", completer.CompleteClusterObjects)
	_ = cmd.MarkFlagRequired("cluster-object")
	_ = cmd.MarkFlagRequired("pipeline")

	return cmd
}
