package operator

import (
	"fmt"
	"os"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestAddImage(t *testing.T) {
	t.Run("Successful replacement", func(t *testing.T) {
		coreDockerImage := "calyptia/core-operator"
		coreInstanceVersion := "v1.0.0"
		file := "image: ghcr.io/calyptia/core-operator:v0.1.0\n"
		expected := "image: calyptia/core-operator:v1.0.0\n"

		result, err := addImage(coreDockerImage, coreInstanceVersion, file)
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}
		if result != expected {
			t.Errorf("Expected: %s, but got: %s", expected, result)
		}
	})

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
}

func TestPrepareManifest(t *testing.T) {
	// Test case setup
	coreInstanceVersion := "v1.0.0"
	coreDockerImage := "calyptia/core-operator"
	namespace := "my-namespace"
	enableExternalTrafficPolicyLocal := true

	t.Run("Successful manifest preparation", func(t *testing.T) {
		// Test the prepareManifest function
		resultFile, err := prepareInstallManifest(coreDockerImage, coreInstanceVersion, namespace, false, enableExternalTrafficPolicyLocal)
		// Verify the results
		if err != nil {
			t.Errorf("Expected no error, but got: %v", err)
		}

		actualFileContents, _ := os.ReadFile(resultFile)

		result := string(actualFileContents)
		assert.Contains(t, result, fmt.Sprintf("image: %s:%s", coreDockerImage, coreInstanceVersion))
		assert.Contains(t, result, fmt.Sprintf("namespace: %s", namespace))
		assert.Contains(t, result, "args: ['"+EnableExternalTrafficPolicyLocal+"']")

		// Clean up the temporary file
		os.Remove(resultFile)
	})
}
