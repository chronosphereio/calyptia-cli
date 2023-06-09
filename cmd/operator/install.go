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
				return fmt.Errorf("could not apply kubernetes manifest: %w", err)
			}
			for _, resource := range resourcesCreated {
				fmt.Printf("%s=%s\n", resource[0], resource[1])
			}
			return nil
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&coreInstanceVersion, "version", "", "Core instance version")
	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	return cmd
}
