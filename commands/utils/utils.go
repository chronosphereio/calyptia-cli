package utils

const (
	LatestVersion                  = "latest"
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
