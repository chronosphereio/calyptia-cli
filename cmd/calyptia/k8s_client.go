package main

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	cloud "github.com/calyptia/api/types"
)

type k8sClient struct {
	kubernetes.Interface
	namespace    string
	projectID    string
	projectToken string
	cloudBaseURL string
}

func (client *k8sClient) ensureOwnNamespace(ctx context.Context) error {
	exists, err := client.ownNamespaceExists(ctx)
	if err != nil {
		return fmt.Errorf("exists: %w", err)
	}

	if exists {
		return nil
	}

	_, err = client.createOwnNamespace(ctx)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil
}

func (client *k8sClient) ownNamespaceExists(ctx context.Context) (bool, error) {
	_, err := client.CoreV1().Namespaces().Get(ctx, client.namespace, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (client *k8sClient) createOwnNamespace(ctx context.Context) (*apiv1.Namespace, error) {
	return client.CoreV1().Namespaces().Create(ctx, &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: client.namespace,
		},
	}, metav1.CreateOptions{})
}

func (client *k8sClient) createClusterRole(ctx context.Context, agg cloud.CreatedAggregator) (*rbacv1.ClusterRole, error) {
	return client.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: agg.Name + "-cluster-role",
			Labels: map[string]string{
				"calyptia_project_id":    client.projectID,
				"calyptia_aggregator_id": agg.ID,
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
	}, metav1.CreateOptions{})
}

func (client *k8sClient) createServiceAccount(ctx context.Context, agg cloud.CreatedAggregator) (*apiv1.ServiceAccount, error) {
	return client.CoreV1().ServiceAccounts(client.namespace).Create(ctx, &apiv1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: agg.Name + "-service-account",
			Labels: map[string]string{
				"calyptia_project_id":    client.projectID,
				"calyptia_aggregator_id": agg.ID,
			},
		},
	}, metav1.CreateOptions{})
}

func (client *k8sClient) createClusterRoleBinding(
	ctx context.Context,
	agg cloud.CreatedAggregator,
	clusterRole *rbacv1.ClusterRole,
	serviceAccount *apiv1.ServiceAccount,
) (*rbacv1.ClusterRoleBinding, error) {
	return client.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: agg.Name + "-cluster-role-binding",
			Labels: map[string]string{
				"calyptia_project_id":    client.projectID,
				"calyptia_aggregator_id": agg.ID,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: client.namespace,
				Name:      serviceAccount.Name,
			},
		},
	}, metav1.CreateOptions{})
}

func (client *k8sClient) createDeployment(
	ctx context.Context,
	agg cloud.CreatedAggregator,
	serviceAccount *apiv1.ServiceAccount,
) (*appsv1.Deployment, error) {
	labels := map[string]string{
		"calyptia_project_id":    client.projectID,
		"calyptia_aggregator_id": agg.ID,
	}
	return client.AppsV1().Deployments(client.namespace).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   agg.Name + "-deployment",
			Labels: labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName:           serviceAccount.Name,
					AutomountServiceAccountToken: ptr(true),
					Containers: []apiv1.Container{
						{
							Name:            agg.Name,
							Image:           "ghcr.io/calyptia/core",
							ImagePullPolicy: apiv1.PullAlways,
							Args:            []string{"-debug=true"},
							Env: []apiv1.EnvVar{
								{
									Name:  "AGGREGATOR_NAME",
									Value: agg.Name,
								},
								{
									Name:  "PROJECT_TOKEN",
									Value: client.projectToken,
								},
								{
									Name:  "AGGREGATOR_FLUENTBIT_CLOUD_URL",
									Value: client.cloudBaseURL,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})
}
