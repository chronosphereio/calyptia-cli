package coreinstance

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/completer"
	cfg "github.com/calyptia/cli/config"
	"github.com/calyptia/cli/confirm"
	"github.com/calyptia/cli/k8s"
)

const (
	itemToDeleteFormat    = "\tnamespace=%s name=%s\n"
	clusterLevelNamespace = "cluster"
)

func NewCmdDeleteCoreInstanceK8s(config *cfg.Config, testClientSet kubernetes.Interface) *cobra.Command {
	var skipError, confirmed bool
	var environment string
	completer := completer.Completer{Config: config}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	cmd := &cobra.Command{
		Use:               "kubernetes CORE_INSTANCE",
		Aliases:           []string{"kube", "k8s"},
		Short:             "Delete a core instance and all of its kubernetes resources",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completer.CompleteCoreInstances,
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

			coreInstanceKey := args[0]
			coreInstanceID, err := completer.LoadCoreInstanceID(coreInstanceKey, environmentID)
			if err != nil {
				return err
			}

			if !confirmed {
				cmd.Printf("Are you sure you want to delete core instance with id %q and all of its associated kubernetes resources? (y/N) ", coreInstanceID)
				confirmed, err := confirm.ReadConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			agg, err := config.Cloud.CoreInstance(ctx, coreInstanceID)
			if err != nil {
				return err
			}

			err = config.Cloud.DeleteCoreInstance(ctx, agg.ID)
			if err != nil {
				return err
			}

			cmd.Printf("Successfully deleted core instance with id %q\n", agg.ID)

			if configOverrides.Context.Namespace == "" {
				configOverrides.Context.Namespace = apiv1.NamespaceDefault
			}

			var clientset kubernetes.Interface
			if testClientSet != nil {
				clientset = testClientSet
			} else {
				kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
				kubeClientConfig, err := kubeConfig.ClientConfig()
				if err != nil {
					return err
				}

				clientset, err = kubernetes.NewForConfig(kubeClientConfig)
				if err != nil {
					return err
				}
			}

			k8sClient := &k8s.Client{
				Interface:    clientset,
				Namespace:    configOverrides.Context.Namespace,
				ProjectToken: config.ProjectToken,
				CloudBaseURL: config.BaseURL,
			}

			label := fmt.Sprintf("%s=%s", k8s.LabelAggregatorID, agg.ID)
			itemsToDelete, err := listDeletionsByLabel(ctx, k8sClient, cmd, label)
			if err != nil {
				return err
			}

			if itemsToDelete == 0 {
				cmd.Println("No kubernetes resources to delete")
				return nil
			}

			namespaces, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
			if err != nil {
				return err
			}

			var deletedInNamespace bool
			for _, ns := range namespaces.Items {
				err = k8sClient.DeleteDeploymentByLabel(ctx, label, ns.Name)
				if err != nil {
					if !skipError {
						return err
					} else {
						cmd.PrintErrf("Error: could not delete deployments: %v\n", err)
					}
				}

				err = k8sClient.DeleteDaemonSetByLabel(ctx, label, ns.Name)
				if err != nil {
					if !skipError {
						return err
					} else {
						cmd.PrintErrf("Error: could not delete daemonSets: %v\n", err)
					}
				}

				services, err := k8sClient.FindServicesByLabel(ctx, label, ns.Name)
				if err != nil {
					return err
				}

				for _, item := range services.Items {
					err := k8sClient.DeleteServiceByName(ctx, item.Name, ns.Name)
					if err != nil {
						if !skipError {
							return err
						} else {
							cmd.PrintErrf("Error: could not delete service %q: %v\n", item.Name, err)
						}
					}
				}

				if !deletedInNamespace {
					roleBinding := fmt.Sprintf("%s-%s-cluster-role-binding", agg.Name, agg.EnvironmentName)
					err = k8sClient.DeleteRoleBindingByLabel(ctx, label)
					if err != nil {
						if !skipError {
							return err
						} else {
							cmd.PrintErrf("Error: could not delete cluster role binding %q: %v\n", roleBinding, err)
						}
					}

					clusterRole := fmt.Sprintf("%s-%s-cluster-role", agg.Name, agg.EnvironmentName)
					err = k8sClient.DeleteClusterRoleByLabel(ctx, label)
					if err != nil {
						if !skipError {
							return err
						} else {
							cmd.PrintErrf("Error: could not delete cluster role %q: %v\n", clusterRole, err)
						}
					}

					deletedInNamespace = true
				}

				serviceAcc := fmt.Sprintf("%s-%s-service-account", agg.Name, agg.EnvironmentName)
				err = k8sClient.DeleteServiceAccountByLabel(ctx, label, ns.Name)
				if err != nil {
					if !skipError {
						return err
					} else {
						cmd.PrintErrf("Error: could not delete service account %q: %v\n", serviceAcc, err)
					}
				}

				secret := agg.Name + "-private-rsa.key"
				err = k8sClient.DeleteSecretByLabel(ctx, label, ns.Name)
				if err != nil {
					if !skipError {
						return err
					} else {
						cmd.PrintErrf("Error: could not delete secret %q: %v\n", secret, err)
					}
				}

				err = k8sClient.DeleteConfigMapsByLabel(ctx, label, ns.Name)
				if err != nil {
					if !skipError {
						return err
					} else {
						cmd.PrintErrf("Error: could not delete config map: %v", err)
					}
				}

			}

			cmd.Printf("Successfully deleted %d kubernetes resources\n", itemsToDelete)

			return nil
		},
	}

	isNonInteractive := os.Stdin == nil || !term.IsTerminal(int(os.Stdin.Fd()))

	fs := cmd.Flags()
	fs.BoolVar(&skipError, "skip-error", false, "Skip errors during delete process")
	fs.BoolVarP(&confirmed, "yes", "y", isNonInteractive, "Confirm deletion")
	fs.StringVar(&environment, "environment", "", "Calyptia environment name")

	clientcmd.BindOverrideFlags(configOverrides, fs, clientcmd.RecommendedConfigOverrideFlags("kube-"))
	_ = cmd.RegisterFlagCompletionFunc("environment", completer.CompleteEnvironments)

	return cmd
}

func listDeletionsByLabel(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string) (int, error) {
	namespaces, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	var itemsToDelete int
	for _, ns := range namespaces.Items {
		count, err := listDaemonSets(ctx, k8sClient, cmd, label, ns.Name)
		itemsToDelete += count
		if err != nil {
			return 0, err
		}
		count, err = listDeployments(ctx, k8sClient, cmd, label, ns.Name)
		itemsToDelete += count
		if err != nil {
			return 0, err
		}

		count, err = listServices(ctx, k8sClient, cmd, label, ns.Name)
		itemsToDelete += count
		if err != nil {
			return 0, err
		}

		count, err = listServiceAccounts(ctx, k8sClient, cmd, label, ns.Name)
		itemsToDelete += count
		if err != nil {
			return 0, err
		}

		count, err = listSecrets(ctx, k8sClient, cmd, label, ns.Name)
		itemsToDelete += count
		if err != nil {
			return 0, err
		}

		count, err = listConfigMaps(ctx, k8sClient, cmd, label, ns.Name)
		itemsToDelete += count
		if err != nil {
			return 0, err
		}

	}

	count, err := listRoleBindings(ctx, k8sClient, cmd, label)
	itemsToDelete += count
	if err != nil {
		return 0, err
	}

	count, err = listClusterRoles(ctx, k8sClient, cmd, label)
	itemsToDelete += count
	return itemsToDelete, err
}

func listSecrets(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string, ns string) (int, error) {
	secrets, err := k8sClient.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(secrets.Items) == 0 {
		return 0, nil
	}

	cmd.Println("Secrets:")
	for _, item := range secrets.Items {
		cmd.Printf(itemToDeleteFormat, ns, item.Name)
	}

	return len(secrets.Items), nil
}

func listRoleBindings(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string) (int, error) {
	roleBindings, err := k8sClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(roleBindings.Items) == 0 {
		return 0, nil
	}

	cmd.Println("Role bindings:")
	for _, item := range roleBindings.Items {
		cmd.Printf(itemToDeleteFormat, clusterLevelNamespace, item.Name)
	}

	return len(roleBindings.Items), nil
}

func listServiceAccounts(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string, ns string) (int, error) {
	serviceAccounts, err := k8sClient.CoreV1().ServiceAccounts(ns).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(serviceAccounts.Items) == 0 {
		return 0, nil
	}

	cmd.Println("Service accounts:")
	for _, item := range serviceAccounts.Items {
		cmd.Printf(itemToDeleteFormat, ns, item.Name)
	}

	return len(serviceAccounts.Items), nil
}

func listConfigMaps(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label, ns string) (int, error) {
	configMaps, err := k8sClient.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(configMaps.Items) == 0 {
		return 0, nil
	}

	cmd.Println("ConfigMaps:")
	for _, item := range configMaps.Items {
		cmd.Printf(itemToDeleteFormat, clusterLevelNamespace, item.Name)
	}

	return len(configMaps.Items), nil
}

func listClusterRoles(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string) (int, error) {
	clusterRoles, err := k8sClient.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(clusterRoles.Items) == 0 {
		return 0, nil
	}

	cmd.Println("Cluster roles:")
	for _, item := range clusterRoles.Items {
		cmd.Printf(itemToDeleteFormat, clusterLevelNamespace, item.Name)
	}

	return len(clusterRoles.Items), nil
}

func listServices(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string, ns string) (int, error) {
	services, err := k8sClient.CoreV1().Services(ns).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(services.Items) == 0 {
		return 0, nil
	}

	cmd.Println("Services:")
	for _, item := range services.Items {
		cmd.Printf(itemToDeleteFormat, ns, item.Name)
	}

	return len(services.Items), nil
}

func listDeployments(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string, ns string) (int, error) {
	deployments, err := k8sClient.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}
	if len(deployments.Items) == 0 {
		return 0, nil
	}

	cmd.Println("Deployments:")
	for _, item := range deployments.Items {
		cmd.Printf(itemToDeleteFormat, ns, item.Name)
	}

	return len(deployments.Items), nil
}

func listDaemonSets(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string, ns string) (int, error) {
	daemonSets, err := k8sClient.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return 0, err
	}

	if len(daemonSets.Items) == 0 {
		return 0, nil
	}

	cmd.Println("DaemonSets:")
	for _, item := range daemonSets.Items {
		cmd.Printf(itemToDeleteFormat, ns, item.Name)
	}

	return len(daemonSets.Items), nil
}
