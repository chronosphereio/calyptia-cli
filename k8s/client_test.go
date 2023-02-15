package k8s

import (
	"context"
	"errors"
	"fmt"
	"testing"

	cloud "github.com/calyptia/api/types"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var client Client
var deployment *appsv1.Deployment

func setupSuite(t *testing.T) {
	client = Client{
		Interface: testclient.NewSimpleClientset(),
		LabelsFunc: func() map[string]string {
			return map[string]string{
				LabelVersion:   "version",
				LabelPartOf:    "calyptia",
				LabelManagedBy: "calyptia-cli",
				LabelCreatedBy: "calyptia-cli",
			}
		},
	}

	k, _ := client.CreateDeployment(context.Background(), "test_image", cloud.CreatedCoreInstance{}, &apiv1.ServiceAccount{}, true, true)
	deployment = k
}

func TestEnsureOwnNamespace(t *testing.T) {
	setupSuite(t)
	err := client.EnsureOwnNamespace(context.Background())
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestCreateSecret(t *testing.T) {
	setupSuite(t)
	created := cloud.CreatedCoreInstance{
		PrivateRSAKey: []byte("test"),
	}
	k, err := client.CreateSecret(context.Background(), created)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}

func TestCreateClusterRole(t *testing.T) {
	setupSuite(t)
	k, err := client.CreateClusterRole(context.Background(), cloud.CreatedCoreInstance{}, ClusterRoleOpt{})
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}

func TestCreateServiceAccount(t *testing.T) {
	setupSuite(t)
	k, err := client.CreateServiceAccount(context.Background(), cloud.CreatedCoreInstance{})
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}

func TestCreateClusterRoleBinding(t *testing.T) {
	setupSuite(t)
	k, err := client.CreateClusterRoleBinding(context.Background(), cloud.CreatedCoreInstance{}, &rbacv1.ClusterRole{}, &apiv1.ServiceAccount{})
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}

func TestCreateDeployment(t *testing.T) {
	setupSuite(t)
	k, err := client.CreateDeployment(context.Background(), "test_image", cloud.CreatedCoreInstance{}, &apiv1.ServiceAccount{}, true, true)

	if status := apierrors.APIStatus(nil); errors.As(err, &status) {
		if status.Status().Code != 409 { // already exists
			t.Log(err)
			t.Fail()
		}
	} else {
		t.Log(err)
		t.Fail()
	}

	deployment = k
}

func TestDeleteDeploymentByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteDeploymentByLabel(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteDaemonSetByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteDaemonSetByLabel(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteClusterRoleByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteClusterRoleByLabel(context.Background(), "test")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteServiceAccountByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteServiceAccountByLabel(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteRoleBindingByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteRoleBindingByLabel(context.Background(), "test")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteServiceByName(t *testing.T) {
	setupSuite(t)
	err := client.DeleteServiceByName(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteSecretByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteSecretByLabel(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestDeleteConfigMapsByLabel(t *testing.T) {
	setupSuite(t)
	err := client.DeleteConfigMapsByLabel(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestFindServicesByLabel(t *testing.T) {
	setupSuite(t)
	k, err := client.FindServicesByLabel(context.Background(), "test", "namespace")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}

func TestUpdateDeploymentByLabel(t *testing.T) {
	setupSuite(t)
	var label string

	for k, v := range deployment.Labels {
		label = fmt.Sprintf("%s=%s", k, v)
		break
	}
	err := client.UpdateDeploymentByLabel(context.Background(), label, "newImage", "true")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func TestFindDeploymentByName(t *testing.T) {
	setupSuite(t)

	k, err := client.FindDeploymentByName(context.Background(), deployment.Name)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}

func TestFindDeploymentByLabel(t *testing.T) {
	setupSuite(t)
	k, err := client.FindDeploymentByLabel(context.Background(), "test")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if k == nil {
		t.Fail()
	}
}
