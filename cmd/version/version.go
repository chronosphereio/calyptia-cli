package version

var (
	DefaultCloudURLStr = "https://cloud-api.calyptia.com"
	Version            = "dev" // To be injected at build time:  -ldflags="-X 'github.com/calyptia/cli/cmd/version.Version=xxx'"
)
