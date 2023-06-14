package coreinstance

import (
	"context"
	"errors"
	"fmt"
	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/version"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/k8s"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // register GCP auth provider
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"time"
)

const defaultVersion = "latest"

func newCmdCreateCoreInstanceOperator(config *cfg.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var (
		coreInstanceName         string
		coreInstanceVersion      string
		coreFluentBitDockerImage string
		noHealthCheckPipeline    bool
		enableClusterLogging     bool
		skipServiceCreation      bool
		environment              string
		tags                     []string
		dryRun                   bool
		waitReady                bool
		noTLSVerify              bool
	)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr"},
		Short:   "Setup a new core operator instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if configOverrides.Context.Namespace == "" {
				namespace, err := k8s.GetCurrentContextNamespace()
				if err != nil {
					if errors.Is(err, k8s.ErrNoContext) {
						cmd.Printf("No context is currently set. Using default namespace.\n")
					} else {
						return err
					}
				}
				if namespace != "" {
					configOverrides.Context.Namespace = namespace
				} else {
					configOverrides.Context.Namespace = apiv1.NamespaceDefault
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
			}

			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = completer.LoadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}
			metadata, err := getCoreInstanceMetadata(k8sClient)
			if err != nil {
				return err
			}
			coreInstanceParams := cloud.CreateCoreInstance{
				Name:                   coreInstanceName,
				AddHealthCheckPipeline: !noHealthCheckPipeline,
				ClusterLogging:         enableClusterLogging,
				EnvironmentID:          environmentID,
				Tags:                   tags,
				SkipServiceCreation:    skipServiceCreation,
				Metadata:               metadata,
				Version:                coreInstanceVersion,
			}

			if coreFluentBitDockerImage != "" {
				coreInstanceParams.Image = &coreFluentBitDockerImage
			}

			created, err := config.Cloud.CreateCoreInstance(ctx, coreInstanceParams)
			if err != nil {
				return fmt.Errorf("could not create core instance at calyptia cloud: %w", err)
			}

			labelsFunc := func() map[string]string {
				return map[string]string{
					k8s.LabelVersion:      version.Version,
					k8s.LabelPartOf:       "calyptia",
					k8s.LabelManagedBy:    "calyptia-cli",
					k8s.LabelCreatedBy:    "calyptia-cli",
					k8s.LabelProjectID:    config.ProjectID,
					k8s.LabelAggregatorID: created.ID,
				}
			}
			k8sClient.LabelsFunc = labelsFunc

			var resourcesCreated []k8s.ResourceRollBack
			secret, err := k8sClient.CreateSecretOperatorRSAKey(ctx, created, dryRun)
			if err != nil {
				fmt.Printf("An error occurred while creating the core operator instance. %w Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				fmt.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}
			err = addToRollBack(err, secret.Name, secret, resourcesCreated)
			if err != nil {
				return err
			}

			var clusterRoleOpts k8s.ClusterRoleOpt
			clusterRole, err := k8sClient.CreateClusterRole(ctx, created, dryRun, clusterRoleOpts)
			if err != nil {
				fmt.Printf("An error occurred while creating the core operator instance. %w Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				fmt.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			err = addToRollBack(err, clusterRole.Name, clusterRole, resourcesCreated)
			if err != nil {
				return err
			}

			serviceAccount, err := k8sClient.CreateServiceAccount(ctx, created, dryRun)
			if err != nil {
				fmt.Printf("An error occurred while creating the core operator instance. %w Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				fmt.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			err = addToRollBack(err, serviceAccount.Name, serviceAccount, resourcesCreated)
			if err != nil {
				return err
			}

			binding, err := k8sClient.CreateClusterRoleBinding(ctx, created, clusterRole, serviceAccount, dryRun)
			if err != nil {
				fmt.Printf("An error occurred while creating the core operator instance. %w Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				fmt.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			err = addToRollBack(err, serviceAccount.Name, binding, resourcesCreated)
			if err != nil {
				return err
			}

			if coreInstanceVersion == "" {
				coreInstanceVersion = defaultVersion
			}
			syncDeployment, err := k8sClient.DeployCoreOperatorSync(ctx, coreInstanceVersion, !noTLSVerify, created, serviceAccount.Name)
			if err != nil {
				fmt.Printf("An error occurred while creating the core operator instance. %w Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				fmt.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			if waitReady {
				start := time.Now()
				fmt.Printf("Waiting for core operator to be ready...\n")
				err := k8sClient.WaitReady(ctx, syncDeployment)
				if err != nil {
					return err
				}
				fmt.Printf("Core operator is ready. Took %s\n", time.Since(start))
			}

			err = addToRollBack(err, serviceAccount.Name, syncDeployment, resourcesCreated)
			if err != nil {
				return err
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

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.BoolVar(&noHealthCheckPipeline, "no-health-check-pipeline", false, "Disable health check pipeline creation alongside the core instance")
	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging pipeline creation.")
	fs.BoolVar(&skipServiceCreation, "skip-service-creation", false, "Skip the creation of kubernetes services for any pipeline under this core instance.")
	fs.BoolVar(&dryRun, "dry-run", false, "Passing this value will skip creation of any Kubernetes resources and it will return resources as YAML manifest")
	fs.BoolVar(&noTLSVerify, "no-tls-verify", false, "Disable TLS verification when connecting to Calyptia Cloud API.")

	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringSliceVar(&tags, "tags", nil, "Tags to apply to the core instance")

	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", completer.CompleteCoreContainerVersion)

	return cmd
}

func addToRollBack(err error, name string, obj runtime.Object, resourcesCreated []k8s.ResourceRollBack) error {
	r, err := extractRollBack(name, obj)
	if err != nil {
		return err
	}
	resourcesCreated = append(resourcesCreated, r)
	return nil
}

func extractRollBack(name string, obj runtime.Object) (k8s.ResourceRollBack, error) {
	resource, err := k8s.ExtractGroupVersionResource(obj)
	if err != nil {
		return k8s.ResourceRollBack{}, err
	}
	back := k8s.ResourceRollBack{
		Name: name,
		GVR:  resource,
	}
	return back, err
}

func getCoreInstanceMetadata(k8s *k8s.Client) (cloud.CoreInstanceMetadata, error) {
	var metadata cloud.CoreInstanceMetadata

	info, err := k8s.GetClusterInfo()
	if err != nil {
		return metadata, err
	}

	metadata.Namespace = info.Namespace
	metadata.ClusterVersion = info.Version
	metadata.ClusterPlatform = info.Platform

	return metadata, nil
}
