package k8s

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
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
