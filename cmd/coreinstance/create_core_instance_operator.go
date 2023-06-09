package coreinstance

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // register GCP auth provider
	"k8s.io/client-go/tools/clientcmd"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/version"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/k8s"
	restclient "k8s.io/client-go/rest"
)

func newCmdCreateCoreInstanceOperator(config *cfg.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var coreInstanceName string
	var coreInstanceVersion string
	var coreFluentBitDockerImage string
	var noHealthCheckPipeline bool
	var enableClusterLogging bool
	var skipServiceCreation bool
	var environment string
	var tags []string
	var dryRun bool

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Setup a new core operator instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			coreInstanceParams := cloud.CreateCoreInstance{
				Name:                   coreInstanceName,
				AddHealthCheckPipeline: !noHealthCheckPipeline,
				ClusterLogging:         enableClusterLogging,
				EnvironmentID:          environmentID,
				Tags:                   tags,
				SkipServiceCreation:    skipServiceCreation,
			}

			if coreFluentBitDockerImage != "" {
				coreInstanceParams.Image = &coreFluentBitDockerImage
			}

			created, err := config.Cloud.CreateCoreInstance(ctx, coreInstanceParams)
			if err != nil {
				return fmt.Errorf("could not create core instance at calyptia cloud: %w", err)
			}

			if configOverrides.Context.Namespace == "" {
				namespace, err := k8s.GetCurrentContextNamespace()
				if err != nil {
					if errors.Is(err, k8s.ErrNoContext) {
						cmd.Printf("No context is currently set. Using default namespace.\n")
						configOverrides.Context.Namespace = apiv1.NamespaceDefault
					} else {
						return err
					}
				} else {
					configOverrides.Context.Namespace = namespace
				}
			}

			var clientSet kubernetes.Interface
			var kubeClientConfig *restclient.Config
			if testClientSet != nil {
				clientSet = testClientSet
			} else {
				var err error
				kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
				kubeClientConfig, err = kubeConfig.ClientConfig()
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
				Config:       kubeClientConfig,
				LabelsFunc: func() map[string]string {
					return map[string]string{
						k8s.LabelVersion:      version.Version,
						k8s.LabelPartOf:       "calyptia",
						k8s.LabelManagedBy:    "calyptia-cli",
						k8s.LabelCreatedBy:    "calyptia-cli",
						k8s.LabelProjectID:    config.ProjectID,
						k8s.LabelAggregatorID: created.ID,
					}
				},
			}

			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}

			secret, err := k8sClient.CreateSecretOperatorRSAKey(ctx, created, dryRun)
			if err != nil {
				return fmt.Errorf("could not create kubernetes secret from private key: %w", err)
			}

			var clusterRoleOpts k8s.ClusterRoleOpt
			clusterRole, err := k8sClient.CreateClusterRole(ctx, created, dryRun, clusterRoleOpts)
			if err != nil {
				return fmt.Errorf("could not create kubernetes cluster role: %w", err)
			}

			serviceAccount, err := k8sClient.CreateServiceAccount(ctx, created, dryRun)
			if err != nil {
				return fmt.Errorf("could not create kubernetes service account: %w", err)
			}

			binding, err := k8sClient.CreateClusterRoleBinding(ctx, created, clusterRole, serviceAccount, dryRun)
			if err != nil {
				return fmt.Errorf("could not create kubernetes cluster role binding: %w", err)
			}

			syncDeployment, err := k8sClient.DeployCoreOperatorSync(ctx, created, serviceAccount.Name)
			if err != nil {
				return fmt.Errorf("could not create kubernetes deployment: %w", err)
			}

			fmt.Printf("Core instance created successfully\n")
			fmt.Printf("Resources created:\n")

			fmt.Printf("Deployment=%s\n", syncDeployment.Name)
			fmt.Printf("Secret=%s\n", secret.Name)
			fmt.Printf("ClusterRole=%s\n", clusterRole.Name)
			fmt.Printf("ClusterRoleBinding=%s\n", binding.Name)
			fmt.Printf("ServiceAccount=%s\n", serviceAccount.Name)

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version")
	fs.StringVar(&coreInstanceName, "name", "", "Core instance name (autogenerated if empty)")
	fs.StringVar(&coreFluentBitDockerImage, "fluent-bit-image", "", "Calyptia core fluent-bit image to use.")

	fs.BoolVar(&noHealthCheckPipeline, "no-health-check-pipeline", false, "Disable health check pipeline creation alongside the core instance")
	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging pipeline creation.")
	fs.BoolVar(&skipServiceCreation, "skip-service-creation", false, "Skip the creation of kubernetes services for any pipeline under this core instance.")
	fs.BoolVar(&dryRun, "dry-run", false, "Passing this value will skip creation of any Kubernetes resources and it will return resources as YAML manifest")

	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringSliceVar(&tags, "tags", nil, "Tags to apply to the core instance")

	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", completer.CompleteCoreContainerVersion)

	return cmd
}
