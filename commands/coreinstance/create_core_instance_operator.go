package coreinstance

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	kschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // register GCP auth provider
	krest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	cloudtypes "github.com/calyptia/api/types"
	"github.com/calyptia/cli/commands/version"
	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/coreversions"
	"github.com/calyptia/cli/formatters"
	"github.com/calyptia/cli/k8s"
)

func newCmdCreateCoreInstanceOperator(cfg *config.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var (
		coreInstanceName                           string
		coreCloudURL                               string
		coreFluentBitDockerImage                   string
		coreDockerToCloudImage                     string
		coreDockerFromCloudImage                   string
		noHealthCheckPipeline                      bool
		healthCheckPipelinePort                    string
		healthCheckPipelineServiceType             string
		enableClusterLogging                       bool
		skipServiceCreation                        bool
		environment                                string
		tags                                       []string
		dryRun                                     bool
		waitReady                                  bool
		waitTimeout                                time.Duration
		noTLSVerify                                bool
		metricsPort                                string
		metrics                                    bool
		cloudProxy, httpProxy, httpsProxy, noProxy string
		memoryLimit                                string
		annotations                                string
		tolerations                                string
	)

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"opr", "k8s"},
		Short:   "Setup a new core instance on top of a operator installation",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
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
					configOverrides.Context.Namespace = corev1.NamespaceDefault
				}
			}
			var clientSet kubernetes.Interface
			var kubeClientConfig *krest.Config
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

			if coreCloudURL == "" {
				coreCloudURL = cfg.BaseURL
			}

			k8sClient := &k8s.Client{
				Interface:    clientSet,
				Namespace:    configOverrides.Context.Namespace,
				ProjectToken: cfg.ProjectToken,
				CloudBaseURL: coreCloudURL,
				Config:       kubeClientConfig,
			}

			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}

			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}

			operatorVersion, err := k8sClient.CheckOperatorVersion(ctx)
			if errors.Is(err, k8s.ErrCoreOperatorNotFound) {
				return fmt.Errorf("calyptia core operator not found running in the cluster. Please install the core operator first (calyptia install operator)")
			}
			if err != nil {
				return err
			}

			cmd.Printf("Found calyptia core operator installed, version: %s...\n", operatorVersion)
			metadata, err := getCoreInstanceMetadata(k8sClient)
			if err != nil {
				return err
			}

			metadata.ClusterName, err = getClusterName()
			if err != nil {
				return err
			}

			coreInstanceParams := cloudtypes.CreateCoreInstance{
				Name:                   coreInstanceName,
				AddHealthCheckPipeline: !noHealthCheckPipeline,
				ClusterLogging:         enableClusterLogging,
				EnvironmentID:          environmentID,
				Tags:                   tags,
				SkipServiceCreation:    skipServiceCreation,
				Metadata:               metadata,
			}

			if !noHealthCheckPipeline {
				if healthCheckPipelinePort != "" {
					p, err := strconv.Atoi(healthCheckPipelinePort)
					if err != nil {
						return err
					}
					if !(p > 0 && p <= 65535) {
						return fmt.Errorf("invalid provided pipeline port number, must be > 0 < 65535: %w", err)
					}
					coreInstanceParams.HealthCheckPipelinePort = uint(p)
				}

				if healthCheckPipelineServiceType != "" {
					if !formatters.ValidPortKind(healthCheckPipelineServiceType) {
						return fmt.Errorf("invalid health check pipeline service type: %s", healthCheckPipelineServiceType)
					}
					coreInstanceParams.HealthCheckPipelinePortKind = cloudtypes.PipelinePortKind(healthCheckPipelineServiceType)
				}
			}

			// Only set the version if != latest, otherwise use the default value
			// for registering this core instance.
			if operatorVersion != coreversions.Latest {
				coreInstanceParams.Version = operatorVersion
			}

			if coreFluentBitDockerImage != "" {
				coreInstanceParams.Image = &coreFluentBitDockerImage
			}

			created, err := cfg.Cloud.CreateCoreInstance(ctx, coreInstanceParams)
			if err != nil {
				return fmt.Errorf("could not create core instance (%q) at calyptia cloud (%q): %w", coreInstanceName, coreCloudURL, err)
			}

			labelsFunc := func() map[string]string {
				return map[string]string{
					k8s.LabelVersion:      version.Version,
					k8s.LabelPartOf:       "calyptia",
					k8s.LabelComponent:    "operator",
					k8s.LabelManagedBy:    "calyptia-cli",
					k8s.LabelCreatedBy:    "calyptia-cli",
					k8s.LabelProjectID:    cfg.ProjectID,
					k8s.LabelAggregatorID: created.ID,
					k8s.LabelInstance:     coreInstanceName,
				}
			}

			k8sClient.LabelsFunc = labelsFunc

			var resourcesCreated []k8s.ResourceRollBack
			secret, err := k8sClient.CreateSecretOperatorRSAKey(ctx, created, dryRun)
			if err != nil {
				cmd.PrintErrf("An error occurred while creating the core operator instance. %s Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				cmd.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			if secret != nil {
				err = addToRollBack(err, secret.Name, configOverrides.Context.Namespace, "secret", &resourcesCreated)
				if err != nil {
					return err
				}
			}
			var clusterRoleOpts k8s.ClusterRoleOpt
			clusterRole, err := k8sClient.CreateClusterRole(ctx, created, dryRun, clusterRoleOpts)
			if err != nil {
				cmd.PrintErrf("An error occurred while creating the core operator instance. %s Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				cmd.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			if clusterRole != nil {
				err = addToRollBack(err, clusterRole.Name, configOverrides.Context.Namespace, "clusterrole", &resourcesCreated)
				if err != nil {
					return err
				}
			}

			serviceAccount, err := k8sClient.CreateServiceAccount(ctx, created, dryRun)
			if err != nil {
				cmd.PrintErrf("An error occurred while creating the core operator instance. %s Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				cmd.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			if serviceAccount != nil {
				err = addToRollBack(err, serviceAccount.Name, configOverrides.Context.Namespace, "serviceaccount", &resourcesCreated)
				if err != nil {
					return err
				}
			}

			binding, err := k8sClient.CreateClusterRoleBinding(ctx, created, clusterRole, serviceAccount, dryRun)
			if err != nil {
				cmd.PrintErrf("An error occurred while creating the core operator instance. %s Rolling back created resources.\n", err)
				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				cmd.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			err = addToRollBack(err, binding.Name, configOverrides.Context.Namespace, "clusterrolebinding", &resourcesCreated)
			if err != nil {
				return err
			}

			if coreDockerToCloudImage == "" {
				coreDockerToCloudImageTag := coreversions.DefaultCoreOperatorToCloudDockerImageTag
				if operatorVersion != "" {
					coreDockerToCloudImageTag = operatorVersion
				}
				coreDockerToCloudImage = fmt.Sprintf("%s:%s", coreversions.DefaultCoreOperatorToCloudDockerImage, coreDockerToCloudImageTag)
			}

			if coreDockerFromCloudImage == "" {
				coreDockerFromCloudImageTag := coreversions.DefaultCoreOperatorFromCloudDockerImageTag
				if operatorVersion != "" {
					coreDockerFromCloudImageTag = operatorVersion
				}
				coreDockerFromCloudImage = fmt.Sprintf("%s:%s", coreversions.DefaultCoreOperatorFromCloudDockerImage, coreDockerFromCloudImageTag)
			}

			syncParams := k8s.DeployCoreOperatorSync{
				CoreCloudURL:        coreCloudURL,
				FromCloudImage:      coreDockerFromCloudImage,
				ToCloudImage:        coreDockerToCloudImage,
				Metrics:             metrics,
				MetricsPort:         metricsPort,
				MemoryLimit:         memoryLimit,
				Annotations:         annotations,
				Tolerations:         tolerations,
				SkipServiceCreation: skipServiceCreation,
				NoTLSVerify:         noTLSVerify,
				CloudProxy:          cloudProxy,
				HTTPProxy:           httpProxy,
				HTTPSProxy:          httpsProxy,
				NoProxy:             noProxy,
				CoreInstance:        created,
				ServiceAccount:      serviceAccount.Name,
			}

			syncDeployment, err := k8sClient.DeployCoreOperatorSync(ctx, syncParams)
			if err != nil {
				if err := cfg.Cloud.DeleteCoreInstance(ctx, created.ID); err != nil {
					return fmt.Errorf("failed to rollback created core instance %v", err)
				}
				cmd.PrintErrf("An error occurred while creating the core operator instance. %s Rolling back created resources.\n", err)

				resources, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not delete resources: %w", err)
				}
				cmd.Printf("Rollback successful. Deleted %d resources.\n", len(resources))
			}

			if waitReady {
				start := time.Now()
				cmd.Printf("Waiting for core instance to be ready...\n")
				err := k8sClient.WaitReady(ctx, syncDeployment.Namespace, syncDeployment.Name, false, waitTimeout)
				if err != nil {
					return err
				}
				cmd.Printf("Core instance is ready. Took %s\n", time.Since(start))
			}

			err = addToRollBack(err, syncDeployment.Name, configOverrides.Context.Namespace, "deployment", &resourcesCreated)
			if err != nil {
				return err
			}

			cmd.Printf("Core instance created successfully\n")
			cmd.Printf("Deployed images=(sync-to-cloud: %s, sync-from-cloud: %s)\n", coreDockerToCloudImage, coreDockerFromCloudImage)
			cmd.Printf("Resources created:\n")

			cmd.Printf("Deployment=%s\n", syncDeployment.Name)
			cmd.Printf("Secret=%s\n", secret.Name)
			cmd.Printf("ClusterRole=%s\n", clusterRole.Name)
			cmd.Printf("ClusterRoleBinding=%s\n", binding.Name)
			cmd.Printf("ServiceAccount=%s\n", serviceAccount.Name)

			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceName, "name", "", "Core instance name (autogenerated if empty)")
	fs.StringVar(&coreFluentBitDockerImage, "fluent-bit-image", "", "Calyptia core fluent-bit image to use.")
	fs.StringVar(&coreCloudURL, "core-cloud-url", cfg.BaseURL, "Override the cloud URL for the core operator instance")

	fs.StringVar(&coreDockerToCloudImage, "image-to-cloud", "", "Calyptia core operator (to-cloud) docker image to use (fully composed docker image).")
	err := fs.MarkHidden("image-to-cloud")
	if err != nil {
		return nil
	}

	fs.StringVar(&coreDockerFromCloudImage, "image-from-cloud", "", "Calyptia core operator (from-cloud) docker image to use (fully composed docker image).")
	err = fs.MarkHidden("image-from-cloud")
	if err != nil {
		return nil
	}

	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.DurationVar(&waitTimeout, "timeout", time.Second*30, "Wait timeout")
	fs.BoolVar(&noHealthCheckPipeline, "no-health-check-pipeline", false, "Disable health check pipeline creation alongside the core instance")
	fs.StringVar(&healthCheckPipelinePort, "health-check-pipeline-port-number", "", "Port number to expose the health-check pipeline")
	fs.StringVar(&healthCheckPipelineServiceType, "health-check-pipeline-service-type", "", fmt.Sprintf("Service type to use for health-check pipeline, options: %s", formatters.PortKinds()))
	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging pipeline creation.")
	fs.BoolVar(&skipServiceCreation, "skip-service-creation", false, "Skip the creation of kubernetes services for any pipeline under this core instance.")
	fs.BoolVar(&dryRun, "dry-run", false, "Passing this value will skip creation of any Kubernetes resources and it will return resources as YAML manifest")
	fs.BoolVar(&noTLSVerify, "no-tls-verify", false, "Disable TLS verification when connecting to Calyptia Cloud API.")
	fs.StringVar(&metricsPort, "metrics-port", "15334", "Port for metrics endpoint.")
	fs.BoolVar(&metrics, "metrics", false, "enable pipeline metrics")
	fs.StringVar(&memoryLimit, "memory-limit", "512Mi", "Minimum memory required")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringVar(&cloudProxy, "cloud-proxy", "", "proxy for cloud api client to use on this core instance")
	fs.StringVar(&httpProxy, "http-proxy", "", "http proxy to use on this core instance")
	fs.StringVar(&noProxy, "no-proxy", "", "no proxy to use on this core instance")
	fs.StringVar(&httpsProxy, "https-proxy", "", "https proxy to use on this core instance")
	fs.StringVar(&annotations, "annotations", "", "Custom annotations for pipelines. Format should be 'annotation1=value1,annotation2=value2'")
	fs.StringVar(&tolerations, "tolerations", "", `Custom tolerations for pipelines. Format should be 'key1=Equal:value1:Execute:3600,key2=Exists:value2:NoExecute`)

	fs.StringSliceVar(&tags, "tags", nil, "Tags to apply to the core instance")

	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	_ = cmd.RegisterFlagCompletionFunc("environment", cfg.Completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", cfg.Completer.CompleteCoreContainerVersion)

	return cmd
}

func addToRollBack(err error, name, namespace, resource string, resourcesCreated *[]k8s.ResourceRollBack) error {
	if err != nil {
		return err
	}

	*resourcesCreated = append(*resourcesCreated, k8s.ResourceRollBack{
		Name:      name,
		Namespace: namespace,
		GVR: kschema.GroupVersionResource{
			Resource: resource,
		},
	})

	return nil
}

func getCoreInstanceMetadata(kclient *k8s.Client) (cloudtypes.CoreInstanceMetadata, error) {
	var metadata cloudtypes.CoreInstanceMetadata

	info, err := kclient.GetClusterInfo()
	if err != nil {
		return metadata, err
	}

	metadata.Namespace = info.Namespace
	metadata.ClusterVersion = info.Version
	metadata.ClusterPlatform = info.Platform

	return metadata, nil
}

func getClusterName() (string, error) {
	var err error
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	rawKubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return "", err
	}
	clusterName := rawKubeConfig.CurrentContext
	if clusterName == "" {
		clusterName = "default"
	}

	return clusterName, nil
}
