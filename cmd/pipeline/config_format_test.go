package pipeline

import (
	"testing"

	cloud "github.com/calyptia/api/types"
)

func TestInferConfigFormat(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		want    cloud.ConfigFormat
		wantErr bool
	}{
		{
			name:    "INI Format",
			file:    "config.ini",
			want:    cloud.ConfigFormatINI,
			wantErr: false,
		},
		{
			name:    "YAML Format",
			file:    "config.yaml",
			want:    cloud.ConfigFormatYAML,
			wantErr: false,
		},
		{
			name:    "YML Format",
			file:    "config.yml",
			want:    cloud.ConfigFormatYAML,
			wantErr: false,
		},
		{
			name:    "Unrecognized Format",
			file:    "config.txt",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InferConfigFormat(tt.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("inferConfigFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("inferConfigFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}
