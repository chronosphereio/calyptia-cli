package pipeline

import (
	"fmt"
	cloud "github.com/calyptia/api/types"
	"path/filepath"
)

func InferConfigFormat(configFile string) (cloud.ConfigFormat, error) {
	switch filepath.Ext(configFile) {
	case ".ini", ".conf":
		return cloud.ConfigFormatINI, nil
	case ".yaml", ".yml":
		return cloud.ConfigFormatYAML, nil
	case ".json":
		return cloud.ConfigFormatJSON, nil
	default:
		return "", fmt.Errorf("unknown configuration file format for file: %q", configFile)
	}
}
