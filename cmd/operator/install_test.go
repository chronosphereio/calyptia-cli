package operator

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/calyptia/cli/k8s"
	"gopkg.in/yaml.v3"

	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
)

func TestAddImage(t *testing.T) {
	t.Run("Successful replacement", func(t *testing.T) {
		coreDockerImage := "calyptia/core-operator"
		coreInstanceVersion := "v1.0.0"
		expected := "calyptia/core-operator:v1.0.0"
		descs := manifests{
			manifest{
				Kind: "Deployment",
				Descriptor: appsv1.Deployment{
					Spec: appsv1.DeploymentSpec{Template: apiv1.PodTemplateSpec{
						Spec: apiv1.PodSpec{
							Containers: []apiv1.Container{
								{Image: " ghcr.io/calyptia/core-operator:v0.1.0"},
							},
						},
					}},
				},
			},
		}

		result, err := addImage(descs, coreDockerImage, coreInstanceVersion)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if result[0].Descriptor.(appsv1.Deployment).Spec.Template.Spec.Containers[0].Image != expected {
			t.Errorf("Expected: %s, but got: %s", expected,
				result[0].Descriptor.(appsv1.Deployment).Spec.Template.Spec.Containers[0].Image)
		}
	})

	/*
		t.Run("No match found", func(t *testing.T) {
			coreDockerImage := "calyptia/core-operator"
			coreInstanceVersion := "v1.0.0"
			file := "name: core-operator\n"

			result, err := addImage(coreDockerImage, coreInstanceVersion, file)
			expectedError := "could not find image in manifest"

			if result != "" {
				t.Errorf("Expected empty result, but got: %s", result)
			}
			if err == nil {
				t.Error("Expected an error, but got no error")
			} else if err.Error() != expectedError {
				t.Errorf("Expected error: %s, but got: %v", expectedError, err)
			}
		})
	*/
}

func TestPrepareManifest(t *testing.T) {
	// Test case setup
	coreInstanceVersion := "v1.0.0"
	coreDockerImage := "calyptia/core-operator"
	namespace := "my-namespace"
	const deploymentManifest string = `
apiVersion: v1
kind: Namespace
metadata:
  labels:
    app.kubernetes.io/component: manager
    app.kubernetes.io/created-by: operator
    app.kubernetes.io/instance: system
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: namespace
    app.kubernetes.io/part-of: operator
    control-plane: controller-manager
  name: calyptia-core
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app.kubernetes.io/component: manager
    app.kubernetes.io/created-by: operator
    app.kubernetes.io/instance: controller-manager
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/name: deployment
    app.kubernetes.io/part-of: operator
    control-plane: controller-manager
  name: controller-manager
  namespace: calyptia-core
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: controller-manager
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
    spec:
      containers:
      - command:
        - /manager
        image: ghcr.io/calyptia/core-operator:v1.0.0-RC1
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 15
          periodSeconds: 20
        name: manager
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          limits:
            cpu: 500m
            memory: 128Mi
          requests:
            cpu: 10m
            memory: 64Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
      securityContext:
        runAsNonRoot: true
      serviceAccountName: controller-manager
      terminationGracePeriodSeconds: 10
`

	t.Run("Successful manifest preparation", func(t *testing.T) {
		// Mocking k8s.GetOperatorManifest
		k8s.GetOperatorManifest = func(version string) ([]byte, error) {
			return []byte(deploymentManifest), nil
		}

		// Test the prepareManifest function
		resultFile, err := prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace, false)
		// Verify the results
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}

		actualFileContents, _ := os.ReadFile(resultFile)

		result := string(actualFileContents)
		if strings.Contains(result, fmt.Sprintf("image: %s:%s", coreDockerImage, coreInstanceVersion)) == false {
			t.Errorf("Expected image: %s:%s, but got: %s", coreDockerImage, coreInstanceVersion, result)
		}
		if strings.Contains(result, fmt.Sprintf("namespace: %s", namespace)) == false {
			t.Errorf("Expected namespace: %s, but got: %s", namespace, result)
		}

		// Clean up the temporary file
		os.Remove(resultFile)
	})
}

func TestParseYAML(t *testing.T) {
	t.Run("test parsing simple example", func(t *testing.T) {
		manifests, err := parseManifest("manifest.yaml")
		if err != nil {
			t.Error(err)
		}

		enc := yaml.NewEncoder(os.Stdout)

		for _, manifest := range *manifests {
			enc.Encode(manifest)
			if err != nil {
				t.Error(err)
			}
		}
	})
}
