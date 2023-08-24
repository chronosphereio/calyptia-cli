package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/calyptia/cli/cmd/utils"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetCurrentContextNamespace(t *testing.T) {
	t.Run("ValidNamespace", testValidNamespace)
	t.Run("NoCurrentContext", testNoCurrentContext)
}

func testValidNamespace(t *testing.T) {
	// Prepare a sample kubeconfig file for testing
	testKubeconfig := `
apiVersion: v1
kind: Config
current-context: test-context
contexts:
- name: test-context
  context:
    namespace: test-namespace
`

	// Create a temporary kubeconfig file for testing
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, ".kube", "config")
	err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0700)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Set the home directory to the temporary directory
	os.Setenv("HOME", tmpDir)

	// Test with a valid kubeconfig
	namespace, err := GetCurrentContextNamespace()
	if err != nil {
		t.Fatal(err)
	}
	if namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", namespace)
	}

	// Clean up the temporary kubeconfig file and reset the home directory
	err = os.RemoveAll(filepath.Dir(kubeconfigPath))
	if err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("HOME")
}

func testNoCurrentContext(t *testing.T) {
	// Prepare a sample kubeconfig file with no current context
	testKubeconfigNoContext := `
apiVersion: v1
kind: Config
contexts: []
`

	// Create a temporary kubeconfig file for testing
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, ".kube", "config")
	err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0700)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(kubeconfigPath, []byte(testKubeconfigNoContext), 0600)
	if err != nil {
		t.Fatal(err)
	}

	// Set the home directory to the temporary directory
	os.Setenv("HOME", tmpDir)

	// Test with no current context
	_, err = GetCurrentContextNamespace()
	if !errors.Is(err, ErrNoContext) {
		t.Errorf("Expected ErrNoContext error, got: %v", err)
	}

	// Clean up the temporary kubeconfig file and reset the home directory
	err = os.RemoveAll(filepath.Dir(kubeconfigPath))
	if err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("HOME")
}

func TestUpdateOperatorDeploymentByLabel(t *testing.T) {
	operatorLabels := map[string]string{
		LabelComponent: "manager",
		LabelCreatedBy: "operator",
		LabelInstance:  "controller-manager",
	}

	dd := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels:    operatorLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: operatorLabels,
			},
			Replicas: nil,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "Test",
						},
					},
				},
			},
		},
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels:    operatorLabels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Image: "Test",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: apiv1.PodRunning,
		},
	}

	tt := []struct {
		name      string
		client    Client
		manager   string
		expectErr bool
	}{
		{
			name: "update operator pass",
			client: Client{
				Interface: fake.NewSimpleClientset(&dd, &pod),
				Namespace: "default"},
			manager:   "manager",
			expectErr: false,
		},
		{
			name: "update operator fail",
			client: Client{
				Interface: fake.NewSimpleClientset(&dd, &pod),
				Namespace: "default"},
			manager:   "manager1",
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			label := fmt.Sprintf("%s=%s,%s=%s,%s=%s", LabelComponent, tc.manager, LabelCreatedBy, "operator", LabelInstance, "controller-manager")

			if err := tc.client.UpdateOperatorDeploymentByLabel(context.TODO(), label, fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorDockerImage, "1234"), false); err != nil && !tc.expectErr {
				t.Errorf("failed to find deployment by label %s", err)
			}
		})
	}
}

func TestUpdateSyncDeploymentByLabel(t *testing.T) {
	syncLabels := map[string]string{
		LabelComponent:    "operator",
		LabelCreatedBy:    "calyptia-cli",
		LabelAggregatorID: "444",
	}
	dd := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels:    syncLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: syncLabels,
			},
			Replicas: nil,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test",
					Labels: syncLabels,
				},

				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "sync-to-cloud",
							Image: "Test",
						},
						{
							Name:  "sync-from-cloud",
							Image: "Test",
						},
					},
				},
			},
		},
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				LabelComponent:    "operator",
				LabelCreatedBy:    "calyptia-cli",
				LabelAggregatorID: "444",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "sync-to-cloud",
					Image: "Test",
				},
				{
					Name:  "sync-from-cloud",
					Image: "Test",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: apiv1.PodRunning,
		},
	}

	tt := []struct {
		name      string
		client    Client
		aggID     string
		expectErr bool
	}{
		{
			name: "update sync pass",
			client: Client{
				Interface: fake.NewSimpleClientset(&dd, &pod),
				Namespace: "default"},
			aggID:     "444",
			expectErr: false,
		},
		{
			name: "update sync fail",
			client: Client{
				Interface: fake.NewSimpleClientset(&dd, &pod),
				Namespace: "default"},
			aggID:     "333",
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			label := fmt.Sprintf("%s=%s,%s=%s,%s=%s", LabelComponent, "operator", LabelCreatedBy, "calyptia-cli", LabelAggregatorID, tc.aggID)

			if err := tc.client.UpdateSyncDeploymentByLabel(context.TODO(), label, fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorDockerImage, "1234"), "true", false); err != nil && !tc.expectErr {
				t.Errorf("failed to find deployment by label %s", err)
			}
		})
	}
}
