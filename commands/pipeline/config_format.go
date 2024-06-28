package pipeline

import (
	"fmt"
	"path/filepath"

	cloudtypes "github.com/calyptia/api/types"
)

func InferConfigFormat(configFile string) (cloudtypes.ConfigFormat, error) {
	switch filepath.Ext(configFile) {
	case ".ini", ".conf":
		return cloudtypes.ConfigFormatINI, nil
	case ".yaml", ".yml":
		return cloudtypes.ConfigFormatYAML, nil
	case ".json":
		return cloudtypes.ConfigFormatJSON, nil
	default:
		return "", fmt.Errorf("unknown configuration file format for file: %q", configFile)
	}
}
