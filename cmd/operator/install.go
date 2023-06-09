package operator

import (
	"context"
	"errors"
	"fmt"
	"github.com/calyptia/cli/cmd/version"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/k8s"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewCmdInstall(config *cfg.Config, testClientSet kubernetes.Interface) *cobra.Command {
	configOverrides := &clientcmd.ConfigOverrides{}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	var coreInstanceVersion string
	var waitReady bool
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
				LabelsFunc: func() map[string]string {
					return map[string]string{
						k8s.LabelVersion:   version.Version,
						k8s.LabelPartOf:    "calyptia",
						k8s.LabelManagedBy: "calyptia-cli",
						k8s.LabelCreatedBy: "calyptia-cli",
					}
				},
			}
			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}

			resourcesCreated, err := k8sClient.DeployOperator(ctx, coreInstanceVersion)
			if err != nil {
				fmt.Printf("could not apply kubernetes manifest, rolling back the following resources:")
				for _, resource := range resourcesCreated {
					fmt.Printf("%s=%s\n", resource.GVR.Resource, resource.Name)
				}
				deleted, err := k8sClient.DeleteResources(ctx, resourcesCreated)
				if err != nil {
					return fmt.Errorf("could not rollback kubernetes manifest: %w", err)
				}
				fmt.Printf("successfully rolled back the following resources:")
				for _, r := range deleted {
					fmt.Printf("%s=%s\n", r.GVR.Resource, r.Name)
				}
			}
			for _, resource := range resourcesCreated {
				fmt.Printf("%s=%s\n", resource.GVR.Resource, resource.Name)
			}
			return nil
		},
	}
	fs := cmd.Flags()
	fs.BoolVar(&waitReady, "wait", false, "Wait for the core instance to be ready before returning")
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	return cmd
}
