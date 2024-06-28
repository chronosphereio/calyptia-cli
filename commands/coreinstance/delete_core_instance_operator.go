package coreinstance

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	krest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/config"
	"github.com/calyptia/cli/k8s"
)

func NewCmdDeleteCoreInstanceOperator(cfg *config.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var (
		confirmed   bool
		environment string
		wait        bool
	)
	configOverrides := &clientcmd.ConfigOverrides{}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	cmd := &cobra.Command{
		Use:     "operator CORE_INSTANCE",
		Aliases: []string{"dcio"},
		Short:   "Delete a core instance operator",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// delete the core instance on the cloud
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = cfg.Completer.LoadEnvironmentID(ctx, environment)
				if err != nil {
					return err
				}
			}
			coreInstanceKey := args[0]
			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey, environmentID)
			if err != nil {
				return err
			}
			coreInstance, err := cfg.Cloud.CoreInstance(ctx, coreInstanceID)
			if err != nil {
				return err
			}

			err = cfg.Cloud.DeleteCoreInstance(ctx, coreInstance.ID)
			if err != nil {
				return err
			}

			// delete the k8s resources
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
			k8sClient := &k8s.Client{
				Interface:    clientSet,
				Namespace:    configOverrides.Context.Namespace,
				ProjectToken: cfg.ProjectToken,
				CloudBaseURL: cfg.BaseURL,
				Config:       kubeClientConfig,
			}

			if err := k8sClient.EnsureOwnNamespace(ctx); err != nil {
				return fmt.Errorf("could not ensure kubernetes namespace exists: %w", err)
			}

			err = k8sClient.DeleteCoreInstance(ctx, coreInstance.Name, coreInstance.EnvironmentName, wait)
			if err != nil {
				return err
			}
			cmd.Printf("Core instance %s deleted\n", coreInstance.Name)
			return nil
		},
	}
	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))
	fs := cmd.Flags()
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")
	fs.BoolVar(&wait, "wait", false, "Wait for the core instance to be deleted")
	return cmd
}
