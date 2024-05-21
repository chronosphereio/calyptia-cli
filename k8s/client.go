package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/component-base/logs"
	kubectl "k8s.io/kubectl/pkg/cmd"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/utils"
)

type objectType string

const (
	deploymentObjectType          objectType = "deployment"
	clusterRoleObjectType         objectType = "cluster-role"
	clusterRoleName               string     = "clusterrole"
	clusterRoleBindingName        string     = "clusterrolebinding"
	clusterRoleBindingObjectType  objectType = "cluster-role-binding"
	secretObjectType              objectType = "secret"
	serviceAccountObjectType      objectType = "service-account"
	coreTLSVerifyEnvVar           string     = "CORE_TLS_VERIFY"
	syncTLSVerifyEnvVar           string     = "NO_TLS_VERIFY"
	syncHttpProxy                 string     = "HTTP_PROXY"
	syncHttpsProxy                string     = "HTTPS_PROXY"
	syncNoProxy                   string     = "NO_PROXY"
	syncCloudProxy                string     = "CLOUD_PROXY"
	coreSkipServiceCreationEnvVar string     = "CORE_INSTANCE_SKIP_SERVICE_CREATION"
	defaultOperatorNamespace                 = "calyptia-core"
	noContainersErrString                    = "no containers found in deployment %s"
)

var (
	ErrNoContext            = fmt.Errorf("no context is currently set")
	ErrCoreOperatorNotFound = fmt.Errorf("could not find core operator across all namespaces")
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
	if apiErrors.IsNotFound(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func (client *Client) createOwnNamespace(ctx context.Context) (*corev1.Namespace, error) {
	return client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: client.Namespace,
		},
	}, metav1.CreateOptions{})
}

// CreateSecret TODO: DELETE AFTER OPERATOR LAUNCHES and create by k8s become deprecated
func (client *Client) CreateSecret(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool) (*corev1.Secret, error) {
	metadata := client.getObjectMeta(agg, secretObjectType)
	req := &corev1.Secret{
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

	sec, err := client.CoreV1().Secrets(client.Namespace).Create(ctx, req, options)
	if err != nil {
		return nil, err
	}

	sec.TypeMeta = req.TypeMeta
	return sec, nil
}

func (client *Client) CreateSecretOperatorRSAKey(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool) (*corev1.Secret, error) {
	metadata := client.getObjectMeta(agg, secretObjectType)
	req := &corev1.Secret{
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

	sec, err := client.CoreV1().Secrets(client.Namespace).Create(ctx, req, options)
	if err != nil {
		return nil, err
	}

	sec.TypeMeta = req.TypeMeta
	return sec, nil
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
		"pods/log",
		"services",
		"configmaps",
		"deployments/scale",
		"secrets",
		"nodes/proxy",
		"nodes",
		"jobs",
		"podsecuritypolicies",
		"ingestchecks",
		"ingestchecks/finalizers",
		"ingestchecks/status",
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

func (client *Client) CreateServiceAccount(ctx context.Context, agg cloud.CreatedCoreInstance, dryRun bool) (*corev1.ServiceAccount, error) {
	req := &corev1.ServiceAccount{
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
	serviceAccount *corev1.ServiceAccount,
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
	coreCloudURL string,
	serviceAccount *corev1.ServiceAccount,
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
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:           serviceAccount.Name,
					AutomountServiceAccountToken: &automountServiceAccountToken,
					Containers: []corev1.Container{
						{
							Name:            agg.Name,
							Image:           image,
							ImagePullPolicy: corev1.PullAlways,
							Args:            []string{"-debug=true"},
							Env: []corev1.EnvVar{
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
									Value: coreCloudURL,
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
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
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
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteDaemonSetByLabel(ctx context.Context, label, ns string) error {
	foreground := metav1.DeletePropagationForeground
	err := client.AppsV1().DaemonSets(ns).DeleteCollection(ctx, metav1.DeleteOptions{
		PropagationPolicy: &foreground,
	}, metav1.ListOptions{LabelSelector: label})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteClusterRoleByLabel(ctx context.Context, label string) error {
	err := client.RbacV1().ClusterRoles().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteServiceAccountByLabel(ctx context.Context, label, ns string) error {
	err := client.CoreV1().ServiceAccounts(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteRoleBindingByLabel(ctx context.Context, label string) error {
	err := client.RbacV1().ClusterRoleBindings().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteServiceByName(ctx context.Context, name, ns string) error {
	err := client.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteSecretByLabel(ctx context.Context, label, ns string) error {
	err := client.CoreV1().Secrets(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) DeleteConfigMapsByLabel(ctx context.Context, label, ns string) error {
	err := client.CoreV1().ConfigMaps(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: label})
	if apiErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (client *Client) FindServicesByLabel(ctx context.Context, label, ns string) (*corev1.ServiceList, error) {
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
		return fmt.Errorf(noContainersErrString, deployment.Name)
	}

	deployment.Spec.Template.Spec.Containers[0].Image = newImage

	envVars := deployment.Spec.Template.Spec.Containers[0].Env

	found := false
	for idx, envVar := range envVars {
		if envVar.Name == coreTLSVerifyEnvVar || envVar.Name == syncTLSVerifyEnvVar {
			if envVar.Value != tlsVerify {
				envVars[idx].Value = tlsVerify
			}
			found = true
		}
	}

	if !found {
		envVars = append(envVars, corev1.EnvVar{
			Name:  syncTLSVerifyEnvVar,
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

func (client *Client) UpdateSyncDeploymentByLabel(ctx context.Context, label, newImage, tlsVerify string, skipServiceCreation, verbose bool, cloudProxy, httpProxy, httpsProxy, noProxy string, waitTimeout time.Duration) error {
	deploymentList, err := client.FindDeploymentByLabel(ctx, label)
	if err != nil {
		return err
	}
	if len(deploymentList.Items) == 0 {
		return fmt.Errorf("no deployment found with label %s", label)
	}
	deployment := deploymentList.Items[0]
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf(noContainersErrString, deployment.Name)
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		if strings.Contains(container.Name, "to-cloud") {
			deployment.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorToCloudDockerImage, newImage)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncTLSVerifyEnvVar, tlsVerify)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncCloudProxy, cloudProxy)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncNoProxy, noProxy)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncHttpProxy, httpProxy)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncHttpsProxy, httpsProxy)
		}
		if strings.Contains(container.Name, "from-cloud") {
			deployment.Spec.Template.Spec.Containers[i].Image = fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorFromCloudDockerImage, newImage)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncTLSVerifyEnvVar, tlsVerify)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncCloudProxy, cloudProxy)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncNoProxy, noProxy)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncHttpProxy, httpProxy)
			deployment.Spec.Template.Spec.Containers[i].Env = client.updateEnvVars(container.Env, syncHttpsProxy, httpsProxy)
			if skipServiceCreation {
				deployment.Spec.Template.Spec.Containers[i].Env = append(deployment.Spec.Template.Spec.Containers[i].Env, corev1.EnvVar{
					Name:  "SKIP_SERVICE_CREATION",
					Value: strconv.FormatBool(true),
				})

			}
		}
	}

	annotations := deployment.Spec.Template.Annotations
	if len(annotations) == 0 {
		annotations = make(map[string]string)
	}
	annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format("20060102150405")
	deployment.Spec.Template.Annotations = annotations

	_, err = client.AppsV1().Deployments(client.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if err := client.WaitReady(ctx, deployment.Namespace, deployment.Name, verbose, waitTimeout); err != nil {
		return err
	}
	return nil
}

func (client *Client) rolloutDeployment(ctx context.Context, namespace, deployment string) error {
	data := fmt.Sprintf(`{"spec": {"template": {"metadata": {"annotations": {"kubectl.kubernetes.io/restartedAt": "%s"}}}}}`, time.Now().Format("20060102150405"))
	_, err := client.AppsV1().Deployments(namespace).Patch(ctx, deployment, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{})

	return err
}

func (client *Client) updateEnvVars(envVars []corev1.EnvVar, key, value string) []corev1.EnvVar {
	found := false
	for idx, envVar := range envVars {
		if envVar.Name == key {
			if envVar.Value != value {
				envVars[idx].Value = value
			}
			found = true
		}
	}

	if !found {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}

	return envVars
}

func (client *Client) UpdateOperatorDeploymentByLabel(ctx context.Context, label string, newImage string, verbose bool, waitTimeout time.Duration) error {
	deploymentList, err := client.FindDeploymentByLabel(ctx, label)
	if err != nil {
		return err
	}
	if len(deploymentList.Items) == 0 {
		return fmt.Errorf("no deployment found with label %s", label)
	}
	deployment := deploymentList.Items[0]
	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf(noContainersErrString, deployment.Name)
	}

	deployment.Spec.Template.Spec.Containers[0].Image = newImage
	_, err = client.AppsV1().Deployments(client.Namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	if err := client.rolloutDeployment(ctx, deployment.Namespace, deployment.Name); err != nil {
		return err
	}

	if err := client.WaitReady(ctx, deployment.Namespace, deployment.Name, verbose, waitTimeout); err != nil {
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

func (client *Client) DeployCoreOperatorSync(ctx context.Context, coreCloudURL, fromCloudImage, toCloudImage string, metricsPort string, memoryLimit string, annotations string, tolerations string, skipServiceCreation, noTLSVerify bool, cloudProxy, httpProxy, httpsProxy, noProxy string, coreInstance cloud.CreatedCoreInstance, serviceAccount string) (*appsv1.Deployment, error) {
	podTolerations, err := validateTolerations(tolerations)
	if err != nil {
		return nil, err
	}

	labels := client.LabelsFunc()
	env := []corev1.EnvVar{
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
			Value: coreCloudURL,
		},
		{
			Name:  "TOKEN",
			Value: client.ProjectToken,
		},
		{
			Name:  "INTERVAL",
			Value: "15s",
		},
		{
			Name:  "NO_TLS_VERIFY",
			Value: strconv.FormatBool(noTLSVerify),
		},
		{
			Name:  "METRICS_PORT",
			Value: metricsPort,
		},
		{
			Name:  "CLOUD_PROXY",
			Value: cloudProxy,
		},
		{
			Name:  "NO_PROXY",
			Value: noProxy,
		},
		{
			Name:  "HTTP_PROXY",
			Value: httpProxy,
		},
		{
			Name:  "HTTPS_PROXY",
			Value: httpsProxy,
		},
		{
			Name:  "ANNOTATIONS",
			Value: annotations,
		},
		{
			Name:  "TOLERATIONS",
			Value: tolerations,
		},
	}

	if skipServiceCreation {
		env = append(env, corev1.EnvVar{
			Name:  "SKIP_SERVICE_CREATION",
			Value: strconv.FormatBool(true),
		})
	}
	toCloud := corev1.Container{
		Name:            coreInstance.Name + "-sync-to-cloud",
		Image:           toCloudImage,
		ImagePullPolicy: corev1.PullAlways,
		Env:             env,
	}

	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{},
	}

	ml, err := resource.ParseQuantity(memoryLimit)
	if err == nil {
		resources.Limits = corev1.ResourceList{
			"memory": ml,
		}
	}

	toCloud.Resources = resources

	fromCloud := corev1.Container{
		Name:            coreInstance.Name + "-sync-from-cloud",
		Image:           fromCloudImage,
		ImagePullPolicy: corev1.PullAlways,
		Env:             env,
		Resources:       resources,
	}

	req := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatResourceName(coreInstance.Name, coreInstance.EnvironmentName, "sync"),
			Namespace: client.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &deploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: serviceAccount,
					Containers:         []corev1.Container{fromCloud, toCloud},
					Tolerations:        podTolerations,
				},
			},
		},
	}

	options := metav1.CreateOptions{}
	return client.AppsV1().Deployments(client.Namespace).Create(ctx, req, options)
}

type ResourceRollBack struct {
	Name      string
	Namespace string
	GVR       schema.GroupVersionResource
}

func (client *Client) DeleteResources(ctx context.Context, resources []ResourceRollBack) ([]ResourceRollBack, error) {
	kctl := newKubectlCmd()

	errs := []error{}
	var deletedResources []ResourceRollBack
	for _, r := range resources {
		if r.GVR.Resource == string(clusterRoleBindingName) || r.GVR.Resource == string(clusterRoleName) {
			kctl.SetArgs([]string{"delete", r.GVR.Resource, r.Name})
		} else {
			kctl.SetArgs([]string{"delete", "-n", r.Namespace, r.GVR.Resource, r.Name})
		}

		err := kctl.Execute()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s, %s/%s", err.Error(), client.Namespace, r.Name))
		}

		deletedResources = append(deletedResources, r)
	}

	if len(errs) > 0 {
		errStr := ""
		for _, e := range errs {
			errStr += e.Error()
		}
		return nil, fmt.Errorf(errStr)
	}
	return deletedResources, nil
}

var GetOperatorManifest = func(version string) ([]byte, error) {
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
	const operatorReleases = "https://api.github.com/repos/chronosphereio/calyptia-core-operator-releases/releases"
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
	kubeconfig := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if kubeconfig == "" {
		kubeconfig = clientcmd.RecommendedHomeFile
	}
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

func (client *Client) WaitReady(ctx context.Context, namespace, name string, verbose bool, waitTimeout time.Duration) error {
	if err := wait.PollUntilContextTimeout(ctx, 3*time.Second, waitTimeout, true, client.isDeploymentReady(namespace, name)); err != nil {
		if verbose {
			get, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(get.Spec.Selector)})
			if err != nil {
				return err
			}

			podMessages := map[string]string{}
			for _, pod := range pods.Items {
				var containerStatus []string
				for _, status := range pod.Status.ContainerStatuses {
					if status.State.Waiting != nil {
						containerStatus = append(containerStatus, status.State.Waiting.Message)
					}
				}
				if len(containerStatus) != 0 {
					podMessages[pod.Name] = strings.Join(containerStatus, "\n")
				}
			}
			if len(podMessages) != 0 {
				var message string
				for k, v := range podMessages {
					message += fmt.Sprintf("* pod %s, Message: %s'\n", k, v)
				}
				return fmt.Errorf("failed while waiting for deployment to start:\n%s", message)
			}
		}
		return err
	}
	return nil
}

func (client *Client) isDeploymentReady(namespace, name string) wait.ConditionWithContextFunc {
	return func(ctx context.Context) (done bool, err error) {
		get, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(get.Spec.Selector)})
		if err != nil {
			return false, err
		}

		var running bool
		for _, pod := range pods.Items {
			running = pod.Status.Phase == corev1.PodRunning
			if !running {
				break
			}
		}
		return running, nil
	}
}

// ClusterInfo information that is retrieved from the running cluster.
type ClusterInfo struct {
	Namespace, Platform, Version string
}

func (client *Client) GetClusterInfo() (ClusterInfo, error) {
	var info ClusterInfo
	serverVersion, err := client.Discovery().ServerVersion()
	if err != nil {
		return info, fmt.Errorf("error getting kubernetes version: %w", err)
	}
	version, err := goversion.NewVersion(serverVersion.String())
	if err != nil {
		return info, fmt.Errorf("could not parse version from kubernetes cluster: %w", err)
	}
	info.Version = version.String()
	info.Namespace = client.Namespace
	info.Platform = serverVersion.Platform
	return info, nil
}

func (client *Client) DeleteCoreInstance(ctx context.Context, name, environment string, shouldWait bool) error {
	core := struct {
		Secret, ServiceAccount, ClusterRole, ClusterRoleBinding, Deployment string
	}{
		Secret:             FormatResourceName(name, environment, "secret"),
		ServiceAccount:     FormatResourceName(name, environment, "service-account"),
		ClusterRole:        FormatResourceName(name, environment, "cluster-role"),
		ClusterRoleBinding: FormatResourceName(name, environment, "cluster-role-binding"),
		Deployment:         FormatResourceName(name, environment, "sync"),
	}

	namespaceList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, namespace := range namespaceList.Items {
		namespaceName := namespace.Name

		// Delete Deployment
		err = client.AppsV1().Deployments(namespaceName).Delete(ctx, core.Deployment, metav1.DeleteOptions{})
		if err != nil && !apiErrors.IsNotFound(err) {
			return err
		}

		// Delete Secret
		err = client.CoreV1().Secrets(namespaceName).Delete(ctx, core.Secret, metav1.DeleteOptions{})
		if err != nil && !apiErrors.IsNotFound(err) {
			return err
		}

		// Delete ClusterRole
		err = client.RbacV1().ClusterRoles().Delete(ctx, core.ClusterRole, metav1.DeleteOptions{})
		if err != nil && !apiErrors.IsNotFound(err) {
			return err
		}

		// Delete ClusterRoleBinding
		err = client.RbacV1().ClusterRoleBindings().Delete(ctx, core.ClusterRoleBinding, metav1.DeleteOptions{})
		if err != nil && !apiErrors.IsNotFound(err) {
			return err
		}

		// Delete ServiceAccount
		err = client.CoreV1().ServiceAccounts(namespaceName).Delete(ctx, core.ServiceAccount, metav1.DeleteOptions{})
		if err != nil && !apiErrors.IsNotFound(err) {
			return err
		}
		if shouldWait {
			// Wait for the resources to be deleted
			err = wait.PollUntilContextTimeout(ctx, time.Second, time.Minute, true, func(ctx context.Context) (bool, error) {
				_, err := client.AppsV1().Deployments(namespaceName).Get(ctx, core.Deployment, metav1.GetOptions{})
				return err != nil, nil
			})
			if err != nil {
				panic(fmt.Errorf("failed to wait for Deployment deletion in namespace %s: %v", namespaceName, err))
			}
		}
	}
	return nil
}

// defaultResourceNamePrefix name prefix to use on objects created on the k8s provider.
const defaultResourceNamePrefix = "calyptia"

// FormatResourceName returns the resource name with a prepended calyptia prefix.
func FormatResourceName(parts ...string) string {
	str := strings.Join(parts, "-")
	if !strings.HasPrefix(str, defaultResourceNamePrefix) {
		return defaultResourceNamePrefix + "-" + str
	}
	return str
}
func (client *Client) CheckOperatorInstalled(ctx context.Context, namespace string) (bool, error) {
	toFind := "calyptia-core-controller-manager"
	if namespace == "" {
		namespace = "default"
	}

	client.Namespace = namespace
	manager, err := client.FindDeploymentByName(ctx, toFind)
	if err != nil {
		return false, err
	}
	managerImage := manager.Spec.Template.Spec.Containers[0].Image
	managerImageVersion := strings.Split(managerImage, ":")[1]
	if managerImageVersion == "" {
		return false, fmt.Errorf("could not parse version from manager image: %s", managerImage)
	}
	return true, nil
}

func (client *Client) CheckOperatorVersion(ctx context.Context) (string, error) {
	manager, err := client.SearchManagerAcrossAllNamespaces(ctx)
	if err != nil {
		return "", err
	}
	managerImage := manager.Spec.Template.Spec.Containers[0].Image
	managerImageVersion := strings.Split(managerImage, ":")[1]
	if managerImageVersion == "" {
		return "", fmt.Errorf("could not parse version from manager image: %s", managerImage)
	}
	return managerImageVersion, nil
}

func (client *Client) SearchManagerAcrossAllNamespaces(ctx context.Context) (*appsv1.Deployment, error) {
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var manager *appsv1.Deployment
	for _, namespace := range namespaces.Items {
		manager, err = client.AppsV1().Deployments(namespace.Name).Get(ctx, "calyptia-core-controller-manager", metav1.GetOptions{})
		if err != nil && !apiErrors.IsNotFound(err) {
			return nil, err
		}
		if manager.Name != "" {
			break
		}
	}
	if manager.Name == "" {
		return nil, ErrCoreOperatorNotFound
	}
	return manager, err
}

// GetNamespace returns the namespace if it exists.
func (client *Client) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (client *Client) IsOperatorInstalled(ctx context.Context) (bool, error) {
	operatorIncomplete := OperatorIncompleteError{
		Errors: []error{},
	}

	dynClient, err := dynamic.NewForConfig(client.Config)
	if err != nil {
		return false, err
	}

	gkv := schema.GroupVersionResource{Group: "core.calyptia.com", Version: "v1", Resource: "pipelines"}
	_, err = dynClient.Resource(gkv).List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("CustomResourceDefinition Pipeline installed"))
	}

	scheme := runtime.NewScheme()
	appsv1.AddToScheme(scheme)
	rbacv1.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	k8sc, err := k8sclient.New(client.Config, k8sclient.Options{Scheme: scheme})
	if err != nil {
		panic(err)
	}
	deploymentList := &appsv1.DeploymentList{}
	if err := k8sc.List(context.Background(), deploymentList, &k8sclient.ListOptions{}); err != nil {
		panic(err)
	}
	for _, i := range deploymentList.Items {
		if i.Name == operatorDeploymentName {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("operator pod: %s/%s", i.Namespace, i.Name))
		}
	}

	clusterRoles := &rbacv1.ClusterRoleList{}
	if err := k8sc.List(context.Background(), clusterRoles, &k8sclient.ListOptions{}); err != nil {
		panic(err)
	}
	for _, i := range clusterRoles.Items {
		if i.Name == "calyptia-core-manager-role" {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ClusterRole: %s", i.Name))
		}
		if i.Name == "calyptia-core-metrics-reader" {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ClusterRole: %s", i.Name))
		}
		if i.Name == "calyptia-core-pod-role" {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ClusterRole: %s", i.Name))
		}
		if i.Name == "calyptia-core-proxy-role" {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ClusterRole: %s", i.Name))
		}
	}

	crbList := &rbacv1.ClusterRoleBindingList{}
	if err := k8sc.List(context.Background(), crbList, &k8sclient.ListOptions{}); err != nil {
		panic(err)
	}

	for _, i := range crbList.Items {
		if i.Name == "calyptia-core-manager-rolebinding" {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ClusterRoleBinding: %s", i.Name))
		}
		if i.Name == "calyptia-core-proxy-rolebinding" {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ClusterRoleBinding: %s", i.Name))
		}
	}

	saList := &corev1.ServiceAccountList{}
	if err := k8sc.List(context.Background(), saList, &k8sclient.ListOptions{}); err != nil {
		panic(err)
	}

	for _, i := range saList.Items {
		if i.Name == operatorDeploymentName {
			operatorIncomplete.Errors = append(operatorIncomplete.Errors, fmt.Errorf("ServiceAccount: %s/%s", i.Namespace, i.Name))
		}
	}

	if len(operatorIncomplete.Errors) > 0 {
		return true, &operatorIncomplete
	}
	return false, nil
}

func (client *Client) PurgeLeftoverRBAC(ctx context.Context) error {

	if err := client.RbacV1().ClusterRoles().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "calyptia.core=core-operator"}); err != nil {
		return err
	}
	if err := client.RbacV1().ClusterRoleBindings().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "calyptia.core=core-operator"}); err != nil {
		return err
	}

	nsList, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range nsList.Items {
		if err := client.CoreV1().ServiceAccounts(ns.Name).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: "calyptia.core=core-operator"}); err != nil {
			return err
		}
	}
	return nil
}

const operatorDeploymentName = "calyptia-core-controller-manager"

type OperatorIncompleteError struct {
	Errors []error
}

func (o *OperatorIncompleteError) Error() string {
	errs := []string{}
	for _, err := range o.Errors {
		errs = append(errs, err.Error())
	}
	return strings.Join(errs, "\n")
}

func validateTolerations(s string) ([]corev1.Toleration, error) {
	if s == "" {
		return nil, nil
	}

	tolerations := []corev1.Toleration{}

	keys := strings.Split(s, ",")
	for _, key := range keys {

		tmp := strings.Split(key, "=")
		if len(tmp) == 1 {
			return nil, fmt.Errorf("no toleration values provided")
		}
		if tmp[0] == "" {
			return nil, fmt.Errorf("no key provided")
		}
		toleration := corev1.Toleration{
			Key: tmp[0],
		}

		values := strings.Split(tmp[1], ":")

		if values[0] != "-" {
			toleration.Operator = corev1.TolerationOperator(values[0])
		}
		if values[1] != "-" {
			toleration.Value = values[1]
		}
		if values[2] != "-" {
			toleration.Effect = corev1.TaintEffect(values[2])
		}

		if len(values) > 3 {
			i, err := strconv.ParseInt(values[3], 10, 64)
			if err == nil {
				toleration.TolerationSeconds = int64Ptr(i)
			} else {
				return nil, err
			}
		}

		tolerations = append(tolerations, toleration)
	}

	for index, toleration := range tolerations {
		if toleration.Operator == "NoExists" && toleration.Value != "" {
			return nil, fmt.Errorf("error: Value cannot be specified for toleration with 'NoExists' operator at index %d", index)
		}

		if toleration.Key == "" && toleration.Operator == "" && toleration.Effect == "" {
			return nil, fmt.Errorf("error: Tolerations at index %d must specify at least one of key, operator, or effect", index)
		}

		if toleration.Key == "" && toleration.Operator == "" {
			return nil, fmt.Errorf("error: Tolerations at index %d must specify key and operator", index)
		}

		if toleration.Key == "" && toleration.Effect == "" {
			return nil, fmt.Errorf("error: Tolerations at index %d must specify key and effect", index)
		}
	}

	return tolerations, nil
}

// Utility function to create a pointer to an int64 value
func int64Ptr(i int64) *int64 {
	return &i
}

func newKubectlCmd() *cobra.Command {
	_ = pflag.CommandLine.MarkHidden("log-flush-frequency")
	_ = pflag.CommandLine.MarkHidden("version")

	args := kubectl.KubectlOptions{
		IOStreams: genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		},
		Arguments: os.Args,
	}

	cmd := kubectl.NewKubectlCommand(args)

	cmd.Aliases = []string{"kc"}
	cmd.Hidden = true
	// Get handle on the original kubectl prerun so we can call it later
	originalPreRunE := cmd.PersistentPreRunE
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Call parents pre-run if exists, cobra does not do this automatically
		// See: https://github.com/spf13/cobra/issues/216
		if parent := cmd.Parent(); parent != nil {
			if parent.PersistentPreRun != nil {
				parent.PersistentPreRun(parent, args)
			}
			if parent.PersistentPreRunE != nil {
				err := parent.PersistentPreRunE(parent, args)
				if err != nil {
					return err
				}
			}
		}
		return originalPreRunE(cmd, args)
	}

	originalRun := cmd.Run
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		originalRun(cmd, args)
		return nil
	}

	logs.AddFlags(cmd.PersistentFlags())
	return cmd
}
