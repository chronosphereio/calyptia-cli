package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlk8s "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"net/http"
	"strings"

	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	cloud "github.com/calyptia/api/types"
)

type objectType string

const (
	deploymentObjectType          objectType = "deployment"
	clusterRoleObjectType         objectType = "cluster-role"
	clusterRoleBindingObjectType  objectType = "cluster-role-binding"
	secretObjectType              objectType = "secret"
	serviceAccountObjectType      objectType = "service-account"
	coreTLSVerifyEnvVar           string     = "CORE_TLS_VERIFY"
	coreSkipServiceCreationEnvVar string     = "CORE_INSTANCE_SKIP_SERVICE_CREATION"
)

var (
	deploymentReplicas           int32 = 1
	automountServiceAccountToken       = true
	defaultObjectMetaNamePrefix        = "calyptia"
)

type Client struct {
	kubernetes.Interface
	Namespace    string
	ProjectToken string
	CloudBaseURL string
	LabelsFunc   func() map[string]string
	Config       *restclient.Config
}

func (client *Client) getObjectMeta(agg cloud.CreatedCoreInstance, objectType objectType) metav1.ObjectMeta {
	name := fmt.Sprintf("%s-%s-%s", agg.Name, agg.EnvironmentName, objectType)
	if !strings.HasPrefix(name, defaultObjectMetaNamePrefix) {
		name = fmt.Sprintf("%s-%s", defaultObjectMetaNamePrefix, name)
	}
	return metav1.ObjectMeta{
		Name:   name,
		Labels: client.LabelsFunc(),
	}
}

func (client *Client) EnsureOwnNamespace(ctx context.Context) error {
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

func (client *Client) ownNamespaceExists(ctx context.Context) (bool, error) {
	_, err := client.CoreV1().Namespaces().Get(ctx, client.Namespace, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (client *Client) createOwnNamespace(ctx context.Context) (*apiv1.Namespace, error) {
	return client.CoreV1().Namespaces().Create(ctx, &apiv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: client.Namespace,
		},
	}, metav1.CreateOptions{})
}

func (client *Client) CreateSecret(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool) (*apiv1.Secret, error) {
	metadata := client.getObjectMeta(agg, secretObjectType)
	req := &apiv1.Secret{

		ObjectMeta: metadata,
		Data: map[string][]byte{
			metadata.Name: agg.PrivateRSAKey,
		},
	}
	req.TypeMeta = metav1.TypeMeta{
		Kind:       "Secret",
		APIVersion: "v1",
	}

	options := metav1.CreateOptions{}
	if dryRun {
		return req, nil
	}

	return client.CoreV1().Secrets(client.Namespace).Create(ctx, req, options)
}

type ClusterRoleOpt struct {
	EnableOpenShift bool
}

func (client *Client) CreateClusterRole(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool, opts ...ClusterRoleOpt) (*rbacv1.ClusterRole, error) {
	apiGroups := []string{"", "apps", "batch", "policy"}
	resources := []string{
		"namespaces",
		"deployments",
		"daemonsets",
		"replicasets",
		"pods",
		"services",
		"configmaps",
		"deployments/scale",
		"secrets",
		"nodes/proxy",
		"nodes",
		"jobs",
		"podsecuritypolicies",
	}

	if len(opts) > 0 {
		enableOpenShift := opts[0].EnableOpenShift
		if enableOpenShift {
			apiGroups = append(apiGroups, "security.openshift.io")
			resources = append(resources, "securitycontextconstraints")
		}
	}
	req := &rbacv1.ClusterRole{
		ObjectMeta: client.getObjectMeta(agg, clusterRoleObjectType),
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: apiGroups,
				Resources: resources,
				Verbs: []string{
					"get",
					"list",
					"create",
					"delete",
					"patch",
					"update",
					"watch",
					"deletecollection",
					"use",
				},
			},
		},
	}

	req.TypeMeta = metav1.TypeMeta{
		Kind:       "ClusterRole",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}

	if dryRun {
		return req, nil
	}

	return client.RbacV1().ClusterRoles().Create(ctx, req, metav1.CreateOptions{})
}

func (client *Client) CreateServiceAccount(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool) (*apiv1.ServiceAccount, error) {
	req := &apiv1.ServiceAccount{

		ObjectMeta: client.getObjectMeta(agg, serviceAccountObjectType),
	}

	req.TypeMeta = metav1.TypeMeta{
		Kind:       "ServiceAccount",
		APIVersion: "v1",
	}

	if dryRun {
		return req, nil
	}

	return client.CoreV1().ServiceAccounts(client.Namespace).Create(ctx, req, metav1.CreateOptions{})
}

func (client *Client) CreateClusterRoleBinding(
	ctx context.Context,
	agg cloud.CreatedCoreInstance,
	clusterRole *rbacv1.ClusterRole,
	serviceAccount *apiv1.ServiceAccount,
	dryRun bool,
) (*rbacv1.ClusterRoleBinding, error) {
	req := &rbacv1.ClusterRoleBinding{
		ObjectMeta: client.getObjectMeta(agg, clusterRoleBindingObjectType),
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRole.Name,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: client.Namespace,
				Name:      serviceAccount.Name,
			},
		},
	}

	req.TypeMeta = metav1.TypeMeta{
		Kind:       "ClusterRoleBinding",
		APIVersion: "rbac.authorization.k8s.io/v1",
	}
	options := metav1.CreateOptions{}
	if dryRun {
		return req, nil
	}

	return client.RbacV1().ClusterRoleBindings().Create(ctx, req, options)
}

func (client *Client) CreateDeployment(
	ctx context.Context,
	image string,
	agg cloud.CreatedCoreInstance,
	serviceAccount *apiv1.ServiceAccount,
	tlsVerify bool,
	skipServiceCreation bool,
	dryRun bool,
) (*appsv1.Deployment, error) {
	labels := client.LabelsFunc()

	req := &appsv1.Deployment{
		ObjectMeta: client.getObjectMeta(agg, deploymentObjectType),
		Spec: appsv1.DeploymentSpec{
			Replicas: &deploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: apiv1.PodSpec{
					ServiceAccountName:           serviceAccount.Name,
					AutomountServiceAccountToken: &automountServiceAccountToken,
					Containers: []apiv1.Container{
						{
							Name:            agg.Name,
							Image:           image,
							ImagePullPolicy: apiv1.PullAlways,
							Args:            []string{"-debug=true"},
							Env: []apiv1.EnvVar{
								{
									Name:  "AGGREGATOR_NAME",
									Value: agg.Name,
								},
								{
									Name:  "PROJECT_TOKEN",
									Value: client.ProjectToken,
								},
								{
									Name:  "AGGREGATOR_FLUENTBIT_CLOUD_URL",
									Value: client.CloudBaseURL,
								},
								{
									Name:  "NATS_URL",
									Value: fmt.Sprintf("nats://tcp-4222-nats-messaging.%s.svc.cluster.local:4222", client.Namespace),
								},
								{
									Name:  coreTLSVerifyEnvVar,
									Value: strconv.FormatBool(tlsVerify),
								},
								{
									Name:  coreSkipServiceCreationEnvVar,
									Value: strconv.FormatBool(skipServiceCreation),
								},
								{
									Name:  "POD_NAMESPACE",
									Value: client.Namespace,
								},
								{
									Name: "DEPLOYMENT_NAME",
									ValueFrom: &apiv1.EnvVarSource{
										FieldRef: &apiv1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	req.TypeMeta = metav1.TypeMeta{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}

	options := metav1.CreateOptions{}
	if dryRun {
		return req, nil
	}

	return client.AppsV1().Deployments(client.Namespace).Create(ctx, req, options)
}

func (client *Client) DeleteDeploymentByLabel(ctx context.Context, label, ns string) error {
	foreground := metav1.DeletePropagationForeground
	err := client.AppsV1().Deployments(ns).DeleteCollection(ctx, metav1.DeleteOptions{
		PropagationPolicy: &foreground,
	}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteDaemonSetByLabel(ctx context.Context, label, ns string) error {
	foreground := metav1.DeletePropagationForeground
	err := client.AppsV1().DaemonSets(ns).DeleteCollection(ctx, metav1.DeleteOptions{
		PropagationPolicy: &foreground,
	}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteClusterRoleByLabel(ctx context.Context, label string) error {
	err := client.RbacV1().ClusterRoles().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteServiceAccountByLabel(ctx context.Context, label, ns string) error {
	err := client.CoreV1().ServiceAccounts(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteRoleBindingByLabel(ctx context.Context, label string) error {
	err := client.RbacV1().ClusterRoleBindings().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteServiceByName(ctx context.Context, name, ns string) error {
	err := client.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteSecretByLabel(ctx context.Context, label, ns string) error {
	err := client.CoreV1().Secrets(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteConfigMapsByLabel(ctx context.Context, label, ns string) error {
	err := client.CoreV1().ConfigMaps(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) FindServicesByLabel(ctx context.Context, label, ns string) (*apiv1.ServiceList, error) {
	return client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: label})
}

func (client *Client) UpdateDeploymentByLabel(ctx context.Context, label, newImage, tlsVerify string) error {
	deploymentList, err := client.FindDeploymentByLabel(ctx, label)
	if err != nil {
		return err
	}
	if len(deploymentList.Items) == 0 {
		return fmt.Errorf("no deployment found with label %s", label)
	}
	deployment := deploymentList.Items[0]
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("no container found in deployment %s", deployment.Name)
	}

	deployment.Spec.Template.Spec.Containers[0].Image = newImage

	envVars := deployment.Spec.Template.Spec.Containers[0].Env

	found := false
	for idx, envVar := range envVars {
		if envVar.Name == coreTLSVerifyEnvVar {
			if envVar.Value != tlsVerify {
				envVars[idx].Value = tlsVerify
			}
			found = true
		}
	}

	if !found {
		envVars = append(envVars, apiv1.EnvVar{
			Name:  coreTLSVerifyEnvVar,
			Value: tlsVerify,
		})
	}

	deployment.Spec.Template.Spec.Containers[0].Env = envVars

	_, err = client.AppsV1().Deployments(client.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (client *Client) FindDeploymentByName(ctx context.Context, name string) (*appsv1.Deployment, error) {
	deployment, err := client.AppsV1().Deployments(client.Namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func (client *Client) FindDeploymentByLabel(ctx context.Context, label string) (*appsv1.DeploymentList, error) {
	return client.AppsV1().Deployments(client.Namespace).List(ctx, metav1.ListOptions{LabelSelector: label})
}

func (client *Client) CreateOperatorSyncDeployment(
	ctx context.Context,
	coreInstance cloud.CreatedCoreInstance,
) (*appsv1.Deployment, error) {
	labels := client.LabelsFunc()

	toCloud := apiv1.Container{

		Name:            coreInstance.Name,
		Image:           "default sync",
		ImagePullPolicy: apiv1.PullAlways,
		Args:            []string{"-debug=true"},
		Env:             []apiv1.EnvVar{},
	}
	fromCloud := apiv1.Container{
		Name:            coreInstance.Name,
		Image:           "default sync",
		ImagePullPolicy: apiv1.PullAlways,
		Args:            []string{"-debug=true"},
		Env:             []apiv1.EnvVar{},
	}

	req := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: client.getObjectMeta(coreInstance, deploymentObjectType),
		Spec: appsv1.DeploymentSpec{
			Replicas: &deploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{fromCloud, toCloud},
				},
			},
		},
	}

	options := metav1.CreateOptions{}

	return client.AppsV1().Deployments(client.Namespace).Create(ctx, req, options)
}

func (client *Client) CreateOperator(ctx context.Context, version string) error {
	manifest, err := getOperatorManifest(version)
	if err != nil {
		return err
	}
	err = applyOperatorManifest(ctx, client.Config, manifest)
	if err != nil {
		return err
	}
	return nil

}

func applyOperatorManifest(ctx context.Context, config *restclient.Config, manifestFull []byte) error {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	manifests := splitManifest(manifestFull)

	var appliedSuccessfully []string
	for _, manifest := range manifests {
		manifest = strings.TrimSpace(manifest)
		if manifest == "" {
			continue
		}

		decoder := yamlk8s.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		obj := &unstructured.Unstructured{}
		_, gvk, err := decoder.Decode([]byte(manifest), nil, obj)
		if err != nil {
			fmt.Printf("Failed to decode manifest: %v", err)
			err := rollbackManifests(ctx, config, appliedSuccessfully)
			return err
		}

		kindPluralized := strings.ToLower(gvk.Kind) + "s"
		resource := dynamicClient.Resource(gvk.GroupVersion().WithResource(kindPluralized))

		created, err := resource.Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			fmt.Printf("Failed to apply manifest: %v", err)
			return err
		}

		appliedSuccessfully = append(appliedSuccessfully, manifest)
		fmt.Printf("Created %s %s\n", created.GetKind(), created.GetName())
	}
	return nil
}

func rollbackManifests(ctx context.Context, config *restclient.Config, manifests []string) error {
	if len(manifests) == 0 {
		return nil
	}
	fmt.Printf("An error ocurred while applying manifests\n")
	fmt.Printf("Trying to rollback %d manifests\n", len(manifests))

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	fmt.Printf("Trying to rollback %d manifests\n", len(manifests))
	for _, manifest := range manifests {
		manifest = strings.TrimSpace(manifest)
		if manifest == "" {
			continue
		}

		decoder := yamlk8s.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		obj := &unstructured.Unstructured{}
		_, gvk, err := decoder.Decode([]byte(manifest), nil, obj)
		if err != nil {
			fmt.Printf("Failed to decode manifest: %v", err)
			continue
		}

		kindPluralized := strings.ToLower(gvk.Kind) + "s"
		resource := dynamicClient.Resource(gvk.GroupVersion().WithResource(kindPluralized))

		err = resource.Namespace(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("Failed to delete manifest: %v", err)
			return err
		}

		fmt.Printf("Deleted %s %s\n", obj.GetKind(), obj.GetName())
	}
	return nil
}

func splitManifest(manifest []byte) []string {
	return strings.Split(string(manifest), "---\n")
}

func getOperatorManifest(version string) ([]byte, error) {
	const operatorReleases = "https://api.github.com/repos/calyptia/core-operator-releases/releases"

	type Release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			BrowserDownloadUrl string `json:"browser_download_url"`
		} `json:"assets"`
	}

	resp, err := http.Get(operatorReleases)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(resp.Body)

	var releases []Release
	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return nil, err
	}

	if version == "" {
		version = "v1.0.0-alpha0"
	}

	var urlForDownload string
	for _, release := range releases {
		if release.TagName == version {
			urlForDownload = release.Assets[0].BrowserDownloadUrl
		}
	}

	response, err := http.Get(urlForDownload)
	if err != nil {
		fmt.Println("Error downloading manifest:", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(response.Body)

	manifestBytes, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading manifest:", err)
		return nil, err
	}

	return manifestBytes, nil
}
