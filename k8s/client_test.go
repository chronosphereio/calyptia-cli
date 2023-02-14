package k8s

import (
	"context"
	"fmt"
	"testing"

	cloud "github.com/calyptia/api/types"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type K8sTestSuite struct {
	suite.Suite
	client     Client
	deployment *appsv1.Deployment
}

func (s *K8sTestSuite) SetupSuite() {
	s.client = Client{
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
}

func (s *K8sTestSuite) TestEnsureOwnNamespace() {
	err := s.client.EnsureOwnNamespace(context.Background())
	s.NoError(err)
}

func (s *K8sTestSuite) TestCreateSecret() {
	created := cloud.CreatedCoreInstance{
		PrivateRSAKey: []byte("test"),
	}
	k, err := s.client.CreateSecret(context.Background(), created)
	s.NoError(err)
	s.NotNil(k)
}

func (s *K8sTestSuite) TestCreateClusterRole() {
	k, err := s.client.CreateClusterRole(context.Background(), cloud.CreatedCoreInstance{}, ClusterRoleOpt{})
	s.NoError(err)
	s.NotNil(k)
}

func (s *K8sTestSuite) TestCreateServiceAccount() {
	k, err := s.client.CreateServiceAccount(context.Background(), cloud.CreatedCoreInstance{})
	s.NoError(err)
	s.NotNil(k)
}

func (s *K8sTestSuite) TestCreateClusterRoleBinding() {
	k, err := s.client.CreateClusterRoleBinding(context.Background(), cloud.CreatedCoreInstance{}, &rbacv1.ClusterRole{}, &apiv1.ServiceAccount{})
	s.NoError(err)
	s.NotNil(k)
}

func (s *K8sTestSuite) TestCreateDeployment() {
	k, err := s.client.CreateDeployment(context.Background(), "test_image", cloud.CreatedCoreInstance{}, &apiv1.ServiceAccount{}, true, true)
	s.NoError(err)
	s.NotNil(k)
	s.deployment = k
}

func (s *K8sTestSuite) TestDeleteDeploymentByLabel() {
	err := s.client.DeleteDeploymentByLabel(context.Background(), "test", "namespace")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteDaemonSetByLabel() {
	err := s.client.DeleteDaemonSetByLabel(context.Background(), "test", "namespace")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteClusterRoleByLabel() {
	err := s.client.DeleteClusterRoleByLabel(context.Background(), "test")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteServiceAccountByLabel() {
	err := s.client.DeleteServiceAccountByLabel(context.Background(), "test", "namespace")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteRoleBindingByLabel() {
	err := s.client.DeleteRoleBindingByLabel(context.Background(), "test")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteServiceByName() {
	err := s.client.DeleteServiceByName(context.Background(), "test", "namespace")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteSecretByLabel() {
	err := s.client.DeleteSecretByLabel(context.Background(), "test", "namespace")
	s.NoError(err)
}

func (s *K8sTestSuite) TestDeleteConfigMapsByLabel() {
	err := s.client.DeleteConfigMapsByLabel(context.Background(), "test", "namespace")
	s.NoError(err)
}

func (s *K8sTestSuite) TestFindServicesByLabel() {
	k, err := s.client.FindServicesByLabel(context.Background(), "test", "namespace")
	s.NoError(err)
	s.NotNil(k)
}

func (s *K8sTestSuite) TestUpdateDeploymentByLabel() {
	var label string

	for k, v := range s.deployment.Labels {
		label = fmt.Sprintf("%s=%s", k, v)
		break
	}
	err := s.client.UpdateDeploymentByLabel(context.Background(), label, "newImage", "true")
	s.NoError(err)
}

func (s *K8sTestSuite) TestFindDeploymentByName() {
	k, err := s.client.FindDeploymentByName(context.Background(), s.deployment.Name)
	s.NoError(err)
	s.NotNil(k)
}

func (s *K8sTestSuite) TestFindDeploymentByLabel() {
	k, err := s.client.FindDeploymentByLabel(context.Background(), "test")
	s.NoError(err)
	s.NotNil(k)
}

func TestK8sTestSuite(t *testing.T) {
	suite.Run(t, new(K8sTestSuite))
}
