package main

import (
	"fmt"
	"sync"

	cloud "github.com/calyptia/api/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newCmdUpdatePipelineSecret(config *config) *cobra.Command {
	return &cobra.Command{
		Use:               "pipeline_secret ID VALUE",
		Short:             "Update a pipeline secret value",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: config.completeSecretIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: update secret by its key. The key is unique per pipeline.
			secretID, value := args[0], args[1]
			err := config.cloud.UpdatePipelineSecret(config.ctx, secretID, cloud.UpdatePipelineSecret{
				Value: ptrBytes([]byte(value)),
			})
			if err != nil {
				return fmt.Errorf("could not update pipeline secret: %w", err)
			}

			return nil
		},
	}
}

func (config *config) completeSecretIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	pipelines, err := config.fetchAllPipelines()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var secrets []cloud.PipelineSecret
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(config.ctx)
	for _, pip := range pipelines {
		pip := pip
		g.Go(func() error {
			ss, err := config.cloud.PipelineSecrets(gctx, pip.ID, cloud.PipelineSecretsParams{})
			if err != nil {
				return err
			}

			mu.Lock()
			secrets = append(secrets, ss...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var uniqueSecretsIDs []string
	secretIDs := map[string]struct{}{}
	for _, s := range secrets {
		if _, ok := secretIDs[s.ID]; !ok {
			uniqueSecretsIDs = append(uniqueSecretsIDs, s.ID)
			secretIDs[s.ID] = struct{}{}
		}
	}

	return uniqueSecretsIDs, cobra.ShellCompDirectiveNoFileComp
}
