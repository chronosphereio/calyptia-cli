package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/calyptia/cli/k8s"
)

const (
	itemToDeleteFormat    = "	- namespace: %s, name: %s"
	clusterLevelNamespace = "cluster"
)

func newCmdDeleteCoreInstanceK8s(config *config, testClientSet kubernetes.Interface) *cobra.Command {
	var skipError, confirmed bool
	var environment string

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	cmd := &cobra.Command{
		Use:               "kubernetes CORE_INSTANCE",
		Aliases:           []string{"kube", "k8s"},
		Short:             "Delete a core instance and all of its kubernetes resources",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAggregators,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			var environmentID string
			if environment != "" {
				var err error
				environmentID, err = config.loadEnvironmentID(environment)
				if err != nil {
					return err
				}
			}

			aggregatorKey := args[0]

			aggregatorID, err := config.loadAggregatorID(aggregatorKey, environmentID)
			if err != nil {
				return err
			}

			if !confirmed {
				cmd.Printf("Are you sure you want to delete core instance with id %q and all of its associated kubernetes resources? (y/N) ", aggregatorID)
				confirmed, err := readConfirm(cmd.InOrStdin())
				if err != nil {
					return err
				}

				if !confirmed {
					cmd.Println("Aborted")
					return nil
				}
			}

			agg, err := config.cloud.Aggregator(ctx, aggregatorID)
			if err != nil {
				return err
			}

			err = config.cloud.DeleteAggregator(ctx, agg.ID)
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
				ProjectToken: config.projectToken,
				CloudBaseURL: config.baseURL,
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
			for _, ns := range namespaces.Items {
				err = k8sClient.DeleteDeploymentByLabel(ctx, label, ns.Name)
				if err != nil {
					if !skipError {
						return err
					} else {
						cmd.PrintErrf("Error: could not delete deployments: %v\n", err)
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

				serviceAcc := agg.Name + "-service-account"
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
			}

			clusterRole := agg.Name + "-cluster-role"
			err = k8sClient.DeleteClusterRoleByLabel(ctx, label)
			if err != nil {
				if !skipError {
					return err
				} else {
					cmd.PrintErrf("Error: could not delete cluster role %q: %v\n", clusterRole, err)
				}
			}

			roleBinding := agg.Name + "-cluster-role-binding"
			err = k8sClient.DeleteRoleBindingByLabel(ctx, label)
			if err != nil {
				if !skipError {
					return err
				} else {
					cmd.PrintErrf("Error: could not delete cluster role binding %q: %v\n", roleBinding, err)
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
	_ = cmd.RegisterFlagCompletionFunc("environment", config.completeEnvironments)

	return cmd
}

func listDeletionsByLabel(ctx context.Context, k8sClient *k8s.Client, cmd *cobra.Command, label string) (int, error) {
	namespaces, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, err
	}

	var itemsToDelete int
	for _, ns := range namespaces.Items {
		count, err := listDeployments(ctx, k8sClient, cmd, label, ns.Name)
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

func ask(rd io.Reader, w io.Writer) bool {
	reader := bufio.NewReader(rd)
	for {
		s, _ := reader.ReadString('\n')
		s = strings.TrimSuffix(s, "\n")
		s = strings.ToLower(s)
		if len(s) > 1 {
			fmt.Fprintln(w, "Please enter Y or N")
			continue
		}
		if strings.Compare(s, "n") == 0 {
			return false
		} else if strings.Compare(s, "y") == 0 {
			break
		} else {
			fmt.Fprintln(w, "Please enter Y or N")
			continue
		}
	}
	return true
}
