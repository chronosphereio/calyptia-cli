package coreinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/itchyny/json2yaml"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // register GCP auth provider
	"k8s.io/client-go/tools/clientcmd"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/utils"
	"github.com/calyptia/cli/cmd/version"
	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/k8s"
)

func newCmdCreateCoreInstanceOnK8s(config *cfg.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var coreInstanceName string
	var coreInstanceVersion string
	var coreDockerImage string
	var coreFluentBitDockerImage string
	var coreCloudURL string
	var noHealthCheckPipeline bool
	var healthCheckPipelinePort string
	var healthCheckPipelineServiceType string
	var noTLSVerify bool
	var enableClusterLogging bool
	var enableOpenShift bool
	var skipServiceCreation bool
	var environment string
	var tags []string
	var dryRun bool

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	completer := completer.Completer{Config: config}

	cmd := &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"kube", "k8s"},
		Short:   "Setup a new core instance on Kubernetes",
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
				return fmt.Errorf("could not create core instance (%q) at calyptia cloud (%q): %w", coreInstanceName, coreCloudURL, err)
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
				ProjectToken: config.ProjectToken,
				CloudBaseURL: config.BaseURL,
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

			secret, err := k8sClient.CreateSecret(ctx, created, dryRun)
			if err != nil {
				return fmt.Errorf("could not create kubernetes secret from private key: %w", err)
			}

			var clusterRoleOpts k8s.ClusterRoleOpt

			clusterRoleOpts.EnableOpenShift = enableOpenShift
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

			if coreDockerImage == "" {
				if coreInstanceVersion != "" {
					coreDockerImage = fmt.Sprintf("%s:%s", utils.DefaultCoreDockerImage, coreInstanceVersion)
				} else {
					coreDockerImage = fmt.Sprintf("%s:%s", utils.DefaultCoreDockerImage, utils.DefaultCoreDockerImageTag)
				}
			}

			if coreCloudURL == "" {
				coreCloudURL = config.BaseURL
			}

			deploy, err := k8sClient.CreateDeployment(ctx, coreDockerImage, created, coreCloudURL,
				serviceAccount, !noTLSVerify, skipServiceCreation, dryRun)
			if err != nil {
				return fmt.Errorf("could not create kubernetes deployment: %w", err)
			}

			if dryRun {
				fmt.Println("---")
				printK8sYaml(secret)
				fmt.Println("---")
				printK8sYaml(clusterRole)
				fmt.Println("---")
				printK8sYaml(serviceAccount)
				fmt.Println("---")
				printK8sYaml(binding)
				fmt.Println("---")
				printK8sYaml(deploy)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "secret=%q\n", secret.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "cluster_role=%q\n", clusterRole.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "service_account=%q\n", serviceAccount.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "cluster_role_binding=%q\n", binding.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "deployment=%q\n", deploy.Name)
			return nil
		},
	}

	fs := cmd.Flags()
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version")
	fs.StringVar(&coreInstanceName, "name", "", "Core instance name (autogenerated if empty)")
	fs.StringVar(&coreDockerImage, "image", "", "Calyptia core docker image to use (fully composed docker image).")
	fs.StringVar(&coreFluentBitDockerImage, "fluent-bit-image", "", "Calyptia core fluent-bit image to use.")
	fs.StringVar(&coreCloudURL, "core-cloud-url", "", "Override the cloud URL for the core instance")

	fs.BoolVar(&noHealthCheckPipeline, "no-health-check-pipeline", false, "Disable health check pipeline creation alongside the core instance")
	fs.StringVar(&healthCheckPipelinePort, "health-check-pipeline-port-number", "", "Port number to expose the health-check pipeline")
	fs.StringVar(&healthCheckPipelineServiceType, "health-check-pipeline-service-type", "", fmt.Sprintf("Service type to use for health-check pipeline, options: %s", AllValidPortKinds()))

	fs.BoolVar(&enableClusterLogging, "enable-cluster-logging", false, "Enable cluster logging pipeline creation.")
	fs.BoolVar(&enableOpenShift, "enable-openshift", false, "Enable Open-Shift specific permissions and settings.")
	fs.BoolVar(&noTLSVerify, "no-tls-verify", false, "Disable TLS verification when connecting to Calyptia Cloud API.")
	fs.BoolVar(&skipServiceCreation, "skip-service-creation", false, "Skip the creation of kubernetes services for any pipeline under this core instance.")
	fs.BoolVar(&dryRun, "dry-run", false, "Passing this value will skip creation of any Kubernetes resources and it will return resources as YAML manifest")

	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.StringSliceVar(&tags, "tags", nil, "Tags to apply to the core instance")

	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))

	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)
	_ = cmd.RegisterFlagCompletionFunc("version", completer.CompleteCoreContainerVersion)

	return cmd
}

// GetK8sYaml strips empty properties which would typically be marshalled with yaml.Marshal
func printK8sYaml(req interface{}) {
	out, _ := json.Marshal(req)
	input := strings.NewReader(string(out))
	var output strings.Builder
	if err := json2yaml.Convert(&output, input); err != nil {
		log.Println("failed to convert JSON to YAML:", err)
		return
	}
	fmt.Println(output.String())
}
