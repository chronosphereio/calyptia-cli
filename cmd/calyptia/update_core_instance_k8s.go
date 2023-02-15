package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/calyptia/utils"
	"github.com/calyptia/cli/k8s"
)

func newCmdUpdateCoreInstanceK8s(config *utils.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var newVersion, newName, environment string
	var (
		disableClusterLogging bool
		enableClusterLogging  bool
		noTLSVerify           bool
		skipServiceCreation   bool
	)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	cmd := &cobra.Command{
		Use:               "kubernetes CORE_INSTANCE",
		Aliases:           []string{"kube", "k8s"},
		Short:             "update a core instance from kubernetes",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.CompleteCoreInstances,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			coreInstanceKey := args[0]

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			coreInstanceID, err := config.LoadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			if coreInstanceKey == newName {
				return fmt.Errorf("cannot update core instance with the same name")
			}

			var opts cloud.UpdateCoreInstance

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

			if skipServiceCreation {
				opts.SkipServiceCreation = &skipServiceCreation
			}

			err = config.Cloud.UpdateCoreInstance(config.Ctx, coreInstanceID, opts)
			if err != nil {
				return fmt.Errorf("could not update core instance at calyptia cloud: %w", err)
			}

			agg, err := config.Cloud.CoreInstance(ctx, coreInstanceID)
			if err != nil {
				return err
			}

			if configOverrides.Context.Namespace == "" {
				configOverrides.Context.Namespace = apiv1.NamespaceDefault
			}

			if newVersion != "" {
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
					ProjectToken: config.ProjectToken,
					CloudBaseURL: config.BaseURL,
				}
				label := fmt.Sprintf("%s=%s,!%s", k8s.LabelAggregatorID, agg.ID, k8s.LabelPipelineID)

				coreDockerImage := fmt.Sprintf("%s:%s", defaultCoreDockerImage, newVersion)

				if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
					return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
				}

				if err := k8sClient.UpdateDeploymentByLabel(ctx, label, coreDockerImage, strconv.FormatBool(!noTLSVerify)); err != nil {
					return fmt.Errorf("could not update kubernetes deployment: %w", err)
				}

				if err != nil {
					return err
				}
				cmd.Printf("calyptia-core instance version updated to version %s\n", newVersion)

			}

			cmd.Printf("calyptia-core instance successfully updated\n")
			return nil
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&newVersion, "version", "", "New version of the calyptia-core instance")
	fs.StringVar(&newName, "name", "", "New core instance name")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging functionality")
	fs.BoolVar(&disableClusterLogging, "disable-cluster-logging", false, "Disable cluster logging functionality")
	fs.BoolVar(&noTLSVerify, "no-tls-verify", false, "Disable TLS verification when connecting to Calyptia Cloud API.")
	fs.BoolVar(&skipServiceCreation, "skip-service-creation", false, "Skip the creation of kubernetes services for any pipeline under this core instance.")

	_ = cmd.RegisterFlagCompletionFunc("environment", config.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", config.CompleteCoreContainerVersion)
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	return cmd
}
