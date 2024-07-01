package resourceprofile

import "encoding/json"

type ResourceProfileSpec struct {
	Resources struct {
		Storage struct {
			SyncFull        bool   `json:"syncFull"`
			BacklogMemLimit string `json:"backlogMemLimit"`
			VolumeSize      string `json:"volumeSize"`
			MaxChunksUp     uint   `json:"maxChunksUp"`
			MaxChunksPause  bool   `json:"maxChunksPause"`
		} `json:"storage"`
		CPU struct {
			BufferWorkers uint   `json:"bufferWorkers"`
			Limit         string `json:"limit"`
			Request       string `json:"request"`
		} `json:"cpu"`
		Memory struct {
			Limit   string `json:"limit"`
			Request string `json:"request"`
		} `json:"memory"`
	} `json:"resources"`
}

var resourceProfileSpecExample = func() string {
	b, err := json.MarshalIndent(ResourceProfileSpec{}, "", "  ")
	if err != nil {
		panic("failed to marshal example spec")
	}

	return string(b)
}()
