// Package coreversions provides the default versions for the core components of the Calyptia platform.
// These values get replaced by CI/CD on every new release.
package coreversions

const (
	Latest                         = "latest"
	DefaultCoreOperatorDockerImage = "ghcr.io/calyptia/core-operator"
	// DefaultCoreOperatorDockerImageTag not manually modified, CI should switch this version on every new release.
	DefaultCoreOperatorDockerImageTag = "v2.14.0"

	DefaultCoreOperatorToCloudDockerImage = "ghcr.io/calyptia/core-operator/sync-to-cloud"
	// DefaultCoreOperatorToCloudDockerImageTag not manually modified, CI should switch this version on every new release.
	DefaultCoreOperatorToCloudDockerImageTag = "v2.14.0"

	DefaultCoreOperatorFromCloudDockerImage = "ghcr.io/calyptia/core-operator/sync-from-cloud"
	// DefaultCoreOperatorFromCloudDockerImageTag not manually modified, CI should switch this version on every new release.
	DefaultCoreOperatorFromCloudDockerImageTag = "v2.14.0"
)
