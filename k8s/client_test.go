package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/calyptia/cli/commands/utils"
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
	err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0o700)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0o600)
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
	err := os.MkdirAll(filepath.Dir(kubeconfigPath), 0o700)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(kubeconfigPath, []byte(testKubeconfigNoContext), 0o600)
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
			Phase: corev1.PodRunning,
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
				Namespace: "default",
			},
			manager:   "manager",
			expectErr: false,
		},
		{
			name: "update operator fail",
			client: Client{
				Interface: fake.NewSimpleClientset(&dd, &pod),
				Namespace: "default",
			},
			manager:   "manager1",
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			label := fmt.Sprintf("%s=%s,%s=%s,%s=%s", LabelComponent, tc.manager, LabelCreatedBy, "operator", LabelInstance, "controller-manager")

			if err := tc.client.UpdateOperatorDeploymentByLabel(context.TODO(), label, fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorDockerImage, "1234"), false, time.Minute); err != nil && !tc.expectErr {
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
			Phase: corev1.PodRunning,
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
				Namespace: "default",
			},
			aggID:     "444",
			expectErr: false,
		},
		{
			name: "update sync fail",
			client: Client{
				Interface: fake.NewSimpleClientset(&dd, &pod),
				Namespace: "default",
			},
			aggID:     "333",
			expectErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			label := fmt.Sprintf("%s=%s,%s=%s,%s=%s", LabelComponent, "operator", LabelCreatedBy, "calyptia-cli", LabelAggregatorID, tc.aggID)
			params := UpdateCoreOperatorSync{
				CloudProxy:          "",
				HttpProxy:           "",
				HttpsProxy:          "",
				NoProxy:             "",
				Image:               fmt.Sprintf("%s:%s", utils.DefaultCoreOperatorDockerImage, "1234"),
				NoTLSVerify:         true,
				SkipServiceCreation: false,
			}
			if err := tc.client.UpdateSyncDeploymentByLabel(context.TODO(), label, params, false, time.Minute); err != nil && !tc.expectErr {
				t.Errorf("failed to find deployment by label %s", err)
			}
		})
	}
}

func TestValidateTolerations(t *testing.T) {
	testCases := []struct {
		input          string
		expectedResult []corev1.Toleration
		expectedError  error
	}{
		// Valid input with single toleration
		{
			input: "key1=Exists:val1:NoSchedule:600",
			expectedResult: []corev1.Toleration{
				{
					Key:               "key1",
					Operator:          corev1.TolerationOpExists,
					Value:             "val1",
					Effect:            corev1.TaintEffectNoSchedule,
					TolerationSeconds: int64Ptr(600),
				},
			},
			expectedError: nil,
		},
		// Valid input with multiple tolerations
		{
			input: "key1=Exists:val1:NoSchedule:600,key2=Equal:val2:PreferNoSchedule:300",
			expectedResult: []corev1.Toleration{
				{
					Key:               "key1",
					Operator:          corev1.TolerationOpExists,
					Value:             "val1",
					Effect:            corev1.TaintEffectNoSchedule,
					TolerationSeconds: int64Ptr(600),
				},
				{
					Key:               "key2",
					Operator:          corev1.TolerationOpEqual,
					Value:             "val2",
					Effect:            corev1.TaintEffectPreferNoSchedule,
					TolerationSeconds: int64Ptr(300),
				},
			},
			expectedError: nil,
		},
		// Invalid input: no toleration values provided
		{
			input:         "key1",
			expectedError: fmt.Errorf("no toleration values provided"),
		},
		// Invalid input: invalid tolerationSeconds value
		{
			input:         "key1=Exists:val1:NoSchedule:invalid",
			expectedError: fmt.Errorf("strconv.ParseInt: parsing \"invalid\": invalid syntax"),
		},
		// Invalid input: invalid tolerationSeconds value
		{
			input:         "key1=Exists:val1:NoSchedule:invalid",
			expectedError: fmt.Errorf("strconv.ParseInt: parsing \"invalid\": invalid syntax"),
		},
		// Invalid input: missing key
		{
			input:         "=Exists:val1:NoSchedule:600",
			expectedError: fmt.Errorf("no key provided"),
		},
	}

	for i, tc := range testCases {
		result, err := validateTolerations(tc.input)

		// Check for error
		if err != nil {
			if tc.expectedError == nil {
				t.Errorf("Test case %d: unexpected error: %s", i, err)
			} else if err.Error() != tc.expectedError.Error() {
				t.Errorf("Test case %d: unexpected error message: got %s, want %s", i, err.Error(), tc.expectedError.Error())
			}
			continue
		} else if tc.expectedError != nil {
			t.Errorf("Test case %d: expected error %s, but got nil", i, tc.expectedError.Error())
			continue
		}

		// Check for result
		if len(result) != len(tc.expectedResult) {
			t.Errorf("Test case %d: unexpected result length: got %d, want %d", i, len(result), len(tc.expectedResult))
			continue
		}

		for j := range result {
			if !tolerationEqual(result[j], tc.expectedResult[j]) {
				t.Errorf("Test case %d: unexpected result: got %v, want %v", i, result[j], tc.expectedResult[j])
			}
		}
	}
}

// Utility function to compare two tolerations for equality
func tolerationEqual(t1, t2 corev1.Toleration) bool {
	return t1.Key == t2.Key &&
		t1.Operator == t2.Operator &&
		t1.Value == t2.Value &&
		t1.Effect == t2.Effect &&
		((t1.TolerationSeconds == nil && t2.TolerationSeconds == nil) || (t1.TolerationSeconds != nil && t2.TolerationSeconds != nil && *t1.TolerationSeconds == *t2.TolerationSeconds))
}
