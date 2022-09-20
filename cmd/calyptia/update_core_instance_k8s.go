package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/k8s"
)

func newCmdUpdateCoreInstanceK8s(config *config, testClientSet kubernetes.Interface) *cobra.Command {
	var newVersion, newName, environment string
	var (
		disableClusterLogging bool
		enableClusterLogging  bool
	)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	cmd := &cobra.Command{
		Use:               "kubernetes CORE_INSTANCE",
		Aliases:           []string{"kube", "k8s"},
		Short:             "update a core instance from kubernetes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAggregators,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			aggregatorKey := args[0]

			coreDockerImage := fmt.Sprintf("%s:%s", defaultCoreDockerImage, newVersion)

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			aggregatorID, err := config.loadAggregatorID(aggregatorKey, environmentID)
			if aggregatorKey == newName {
				return fmt.Errorf("cannot update core instance with the same name")
			}

			var opts cloud.UpdateAggregator

			if newName != "" {
				opts.Name = &newName
			}

			if enableClusterLogging && disableClusterLogging {
				return fmt.Errorf("either --enable-cluster-logging or --disable-cluster-logging can be set")
			}

			if enableClusterLogging {
				opts.ClusterLogging = &enableClusterLogging
			} else if disableClusterLogging {
				disableClusterLogging = !disableClusterLogging
				opts.ClusterLogging = &disableClusterLogging
			}

			err = config.cloud.UpdateAggregator(config.ctx, aggregatorID, opts)
			if err != nil {
				return fmt.Errorf("could not update core instance: %w", err)
			}

			agg, err := config.cloud.Aggregator(ctx, aggregatorID)
			if err != nil {
				return err
			}

			if configOverrides.Context.Namespace == "" {
				configOverrides.Context.Namespace = apiv1.NamespaceDefault
			}

			var clientSet kubernetes.Interface
			if testClientSet != nil {
				clientSet = testClientSet
			} else {
				kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
				kubeClientConfig, err := kubeConfig.ClientConfig()
				if err != nil {
					return err
				}

				clientSet, err = kubernetes.NewForConfig(kubeClientConfig)
				if err != nil {
					return err
				}

			}

			k8sClient := &k8s.Client{
				Interface:    clientSet,
				Namespace:    configOverrides.Context.Namespace,
				ProjectToken: config.projectToken,
				CloudBaseURL: config.baseURL,
			}

			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}
			label := fmt.Sprintf("%s=%s,!%s", k8s.LabelAggregatorID, agg.ID, k8s.LabelPipelineID)
			if err := k8sClient.UpdateDeploymentByLabel(ctx, label, coreDockerImage); err != nil {
				return fmt.Errorf("could not update kubernetes deployment: %w", err)
			}

			if err != nil {
				return err
			}
			cmd.Printf("calyptia-core instance %q was successfully updated to version %s\n", agg.Name, newVersion)
			return nil
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&newVersion, "version", "", "New version of the calyptia-core instance")
	fs.StringVar(&newName, "name", "", "New core instance name")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging functionality")
	fs.BoolVar(&disableClusterLogging, "disable-cluster-logging", false, "Disable cluster logging functionality")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", config.completeCoreContainerVersion)
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	return cmd
}
