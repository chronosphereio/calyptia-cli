package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlk8s "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"path/filepath"
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
	defaultOperatorNamespace                 = "calyptia-core"
)

var ErrNoContext = fmt.Errorf("no context is currently set")
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

// TODO: DELETE AFTER OPERATOR LAUNCHES and create by k8s become deprecated
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

func (client *Client) CreateSecretOperatorRSAKey(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool) (*apiv1.Secret, error) {
	metadata := client.getObjectMeta(agg, secretObjectType)
	req := &apiv1.Secret{

		ObjectMeta: metadata,
		Data: map[string][]byte{
			"private-key": agg.PrivateRSAKey,
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
	apiGroups := []string{"", "apps", "batch", "policy", "core.calyptia.com"}
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
		"pipelines",
		"pipelines/finalizers",
		"pipelines/status",
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

func (client *Client) DeployCoreOperatorSync(ctx context.Context, coreInstance cloud.CreatedCoreInstance, serviceAccount string) (*appsv1.Deployment, error) {
	labels := client.LabelsFunc()
	const toCloudImage = "ghcr.io/calyptia/core-operator/sync-to-cloud:v1.0.0-alpha0"
	const fromCloudImage = "ghcr.io/calyptia/core-operator/sync-from-cloud:v1.0.0-alpha0"
	env := []apiv1.EnvVar{
		{
			Name:  "CORE_INSTANCE",
			Value: coreInstance.Name,
		},
		{
			Name:  "NAMESPACE",
			Value: client.Namespace,
		},
		{
			Name:  "CLOUD_URL",
			Value: client.CloudBaseURL,
		},
		{
			Name:  "TOKEN",
			Value: client.ProjectToken,
		},
	}
	toCloud := apiv1.Container{
		Name:            coreInstance.Name + "-sync-to-cloud",
		Image:           toCloudImage,
		ImagePullPolicy: apiv1.PullAlways,
		Env:             env,
	}
	fromCloud := apiv1.Container{
		Name:            coreInstance.Name + "-sync-from-cloud",
		Image:           fromCloudImage,
		ImagePullPolicy: apiv1.PullAlways,
		Env:             env,
	}

	req := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      coreInstance.Name + "-sync",
			Namespace: client.Namespace,
		},
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
					ServiceAccountName: serviceAccount,
					Containers:         []apiv1.Container{fromCloud, toCloud},
				},
			},
		},
	}

	options := metav1.CreateOptions{}

	return client.AppsV1().Deployments(client.Namespace).Create(ctx, req, options)
}

func (client *Client) DeployOperator(ctx context.Context, version string) ([]ResourceRollBack, error) {
	file, err := getOperatorManifest(version)
	if err != nil {
		return nil, err
	}
	applied, err := client.applyOperatorManifest(ctx, file)
	if err != nil {
		return nil, err
	}
	return applied, nil

}

func (client *Client) applyOperatorManifest(ctx context.Context, manifestFull []byte) ([]ResourceRollBack, error) {
	dynamicClient, err := dynamic.NewForConfig(client.Config)
	if err != nil {
		return nil, err
	}

	manifests := splitManifest(manifestFull)

	var appliedSuccessfully []ResourceRollBack
	for _, manifest := range manifests {
		manifest = strings.TrimSpace(manifest)
		if manifest == "" {
			continue
		}

		decoder := yamlk8s.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		obj := &unstructured.Unstructured{}
		_, gvk, err := decoder.Decode([]byte(manifest), nil, obj)
		if err != nil {
			return nil, err
		}

		//workaround to avoid creating a namespace, not needed if we remove the namespace from the manifest
		if gvk.Kind == "Namespace" {
			continue
		}
		namespace := obj.GetNamespace()
		if namespace == defaultOperatorNamespace {
			obj.SetNamespace(client.Namespace)
		}

		kindPluralized := strings.ToLower(gvk.Kind) + "s"
		withResource := gvk.GroupVersion().WithResource(kindPluralized)
		resource := dynamicClient.Resource(withResource)

		//if already exists, skip
		get, err := resource.Namespace(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return nil, err
			}
		}
		if get != nil {
			continue
		}

		created, err := resource.Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		appliedSuccessfully = append(appliedSuccessfully, ResourceRollBack{Name: created.GetName(), GVR: withResource})
	}
	return appliedSuccessfully, nil
}

type ResourceRollBack struct {
	Name string
	GVR  schema.GroupVersionResource
}

func (client *Client) DeleteResources(ctx context.Context, resources []ResourceRollBack) ([]ResourceRollBack, error) {
	dynamicClient, err := dynamic.NewForConfig(client.Config)
	if err != nil {
		return nil, err
	}
	var deletedResources []ResourceRollBack
	for _, r := range resources {
		resource := dynamicClient.Resource(r.GVR)
		err = resource.Namespace(client.Namespace).Delete(ctx, r.Name, metav1.DeleteOptions{})
		if err != nil {
			return nil, err
		}
		deletedResources = append(deletedResources, r)
	}
	return deletedResources, nil
}

func splitManifest(manifest []byte) []string {
	return strings.Split(string(manifest), "---\n")
}

func getOperatorManifest(version string) ([]byte, error) {
	url, err := getOperatorDownloadURL(version)
	if err != nil {
		return nil, err
	}
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error downloading operator manifest: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
		}
	}(response.Body)

	manifestBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return manifestBytes, nil
}

func getOperatorDownloadURL(version string) (string, error) {
	const operatorReleases = "https://api.github.com/repos/calyptia/core-operator-releases/releases"
	type Release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			BrowserDownloadUrl string `json:"browser_download_url"`
		} `json:"assets"`
	}

	resp, err := http.Get(operatorReleases)
	if err != nil {
		return "", fmt.Errorf("failed to get releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var releases []Release
	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return "", fmt.Errorf("failed to decode releases: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	if version == "" {
		if len(releases[0].Assets) == 0 {
			return "", fmt.Errorf("no assets found for the latest release")
		}
		return releases[0].Assets[0].BrowserDownloadUrl, nil
	}

	for _, release := range releases {
		if release.TagName == version {
			if len(release.Assets) == 0 {
				return "", fmt.Errorf("no assets found for the version: %s", version)
			}
			return release.Assets[0].BrowserDownloadUrl, nil
		}
	}

	return "", fmt.Errorf("version %s not found", version)
}

func GetCurrentContextNamespace() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	kubeconfig := filepath.Join(homeDir, ".kube", "config")
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return "", err
	}
	currentContext := config.CurrentContext
	if currentContext == "" {
		return "", ErrNoContext
	}
	context := config.Contexts[currentContext]
	if context == nil {
		return "", ErrNoContext
	}
	return context.Namespace, nil
}

func ExtractGroupVersionResource(obj runtime.Object) (schema.GroupVersionResource, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: gvk.Kind + "s",
	}
	return gvr, nil
}
