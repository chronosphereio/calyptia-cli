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
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/k8s"
)

func NewCmdDeleteCoreInstanceOperator(cfg *config.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var (
		confirmed bool
		wait      bool
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
			if !confirmed {
				cmd.Printf("Are you sure you want to delete core instance %q? (y/N) ", args[0])
				confirmed, err := confirm.Read(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			coreInstanceKey := args[0]
			coreInstanceID, err := cfg.Completer.LoadCoreInstanceID(ctx, coreInstanceKey)
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
	fs.BoolVar(&wait, "wait", false, "Wait for the core instance to be deleted")
	return cmd
}
