package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func newCmdCreateAggregator(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aggregator",
		Short: "Create a new aggregator",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			kubeconfig := filepath.Join(home, ".kube", "config")
			k8sConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return err
			}

			clientset, err := kubernetes.NewForConfig(k8sConfig)
			if err != nil {
				return err
			}

			ctx := context.Background()

			clusterRole, err := config.createClusterRole(ctx, clientset)
			if err != nil {
				return err
			}

			fmt.Printf("create cluster role result: %+v\n", clusterRole.Name)

			serviceAccount, err := config.createServiceAccount(ctx, clientset)
			if err != nil {
				return err
			}

			fmt.Printf("create service account result: %+v\n", serviceAccount.Name)

			binding, err := config.createClusterRoleBinding(ctx, clientset)
			if err != nil {
				return err
			}

			fmt.Printf("create cluster role binding result: %+v\n", binding.Name)

			deploy, err := config.createDeployment(ctx, clientset)
			if err != nil {
				return err
			}

			fmt.Printf("create deploy result: %+v\n", deploy)

			return nil
		},
	}

	return cmd
}

func (config *config) createClusterRole(ctx context.Context, clientset *kubernetes.Clientset) (*rbacv1.ClusterRole, error) {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-cluster-role",
			Labels: map[string]string{
				"app": "demo",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"", "apps"},
				Resources: []string{
					"namespaces",
					"deployments",
					"replicasets",
					"pods",
					"services",
					"configmaps",
					"deployments/scale",
					"secrets",
				},
				Verbs: []string{
					"get",
					"list",
					"create",
					"delete",
					"patch",
					"update",
					"watch",
					"deletecollection",
				},
			},
		},
	}
	return clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
}

func (config *config) createServiceAccount(ctx context.Context, clientset *kubernetes.Clientset) (*apiv1.ServiceAccount, error) {
	serviceAccount := &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-service-account",
			Labels: map[string]string{
				"app": "demo",
			},
			Annotations: map[string]string{
				// TODO
			},
		},
	}
	return clientset.CoreV1().ServiceAccounts("default").Create(ctx, serviceAccount, metav1.CreateOptions{})
}

func (config *config) createClusterRoleBinding(ctx context.Context, clientset *kubernetes.Clientset) (*rbacv1.ClusterRoleBinding, error) {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-cluster-role-binding",
			Labels: map[string]string{
				"app": "demo",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "demo-cluster-role",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "demo-service-account",
				Namespace: "default",
			},
		},
	}
	return clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
}

func (config *config) createDeployment(ctx context.Context, clientset *kubernetes.Clientset) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName:           "demo-service-account",
					AutomountServiceAccountToken: ptr(true),
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "ghcr.io/calyptia/core",
							Args:  []string{"-debug=true"},
							Env: []apiv1.EnvVar{
								{
									Name:  "PROJECT_TOKEN",
									Value: config.projectToken,
								},
								{
									Name:  "AGGREGATOR_FLUENTBIT_CLOUD_URL",
									Value: "https://cloud-api-dev.calyptia.com",
								},
							},
						},
					},
				},
			},
		},
	}
	return clientset.AppsV1().Deployments(apiv1.NamespaceDefault).Create(ctx, deployment, metav1.CreateOptions{})
}

func ptr[T any](p T) *T {
	return &p
}
