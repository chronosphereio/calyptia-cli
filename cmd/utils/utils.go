package utils

import (
	"fmt"
	"math"
	"strings"

	"code.cloudfoundry.org/bytefmt"

	metrics "github.com/calyptia/cli/metric"

	cloud "github.com/calyptia/api/types"
)

const (
	LatestVersion = "latest"
	//nolint: gosec // this is not a secret leak, it's just a format declaration.
	DefaultCoreDockerImage    = "ghcr.io/calyptia/core"
	DefaultCoreDockerImageTag = LatestVersion

	DefaultCoreOperatorDockerImage = "ghcr.io/calyptia/core-operator"
	// DefaultCoreOperatorDockerImageTag not manually modified, CI should switch this version on every new release.
	DefaultCoreOperatorDockerImageTag = "v2.5.1"

	DefaultCoreOperatorToCloudDockerImage = "ghcr.io/calyptia/core-operator/sync-to-cloud"
	// DefaultCoreOperatorToCloudDockerImageTag not manually modified, CI should switch this version on every new release.
	DefaultCoreOperatorToCloudDockerImageTag = "v2.5.1"

	DefaultCoreOperatorFromCloudDockerImage = "ghcr.io/calyptia/core-operator/sync-from-cloud"
	// DefaultCoreOperatorFromCloudDockerImageTag not manually modified, CI should switch this version on every new release.
	DefaultCoreOperatorFromCloudDockerImageTag = "v2.5.1"
)

type RecordCell struct {
	Value *float64
}

func (f RecordCell) String() string {
	if f.Value == nil {
		return ""
	}

	var s string
	if *f.Value > -1 && *f.Value < 1 {
		s = fmt.Sprintf("%.2f", *f.Value)
	} else {
		s = fmt.Sprintf("%.0f", math.Round(*f.Value))
	}
	s = strings.TrimSuffix(s, "0")
	s = strings.TrimSuffix(s, "0")
	s = strings.TrimSuffix(s, ".")
	return s
}

type ByteCell struct {
	Value *float64
}

func (f ByteCell) String() string {
	if f.Value == nil {
		return ""
	}

	s := bytefmt.ByteSize(uint64(math.Round(*f.Value)))
	s = strings.TrimSuffix(s, "B")
	s = strings.ToLower(s)
	return s
}

type Rates struct {
	InputBytes    *float64
	InputRecords  *float64
	OutputBytes   *float64
	OutputRecords *float64
}

func (rates Rates) OK() bool {
	return rates.InputBytes != nil || rates.InputRecords != nil || rates.OutputBytes != nil || rates.OutputRecords != nil
}

func (rates *Rates) Apply(measurement, metric string, points []cloud.MetricFields) {
	if strings.Contains(metric, "dropped_records") {
		return
	}

	if strings.Contains(metric, "retried_records") {
		return
	}

	if strings.Contains(metric, "retries_failed") {
		return
	}

	if strings.Contains(metric, "retries") {
		return
	}

	if strings.Contains(metric, "record") {
		switch measurement {
		case "fluentbit_input", "fluentd_input":
			rates.InputRecords = metrics.Rate(points)
		case "fluentbit_output", "fluentd_output":
			rates.OutputRecords = metrics.Rate(points)
		}
		return
	}

	if strings.Contains(metric, "byte") || strings.Contains(metric, "size") {
		switch measurement {
		case "fluentbit_input", "fluentd_input":
			rates.InputBytes = metrics.Rate(points)
		case "fluentbit_output", "fluentd_output":
			rates.OutputBytes = metrics.Rate(points)
		}
	}
}

func ZeroOfPtr[T comparable](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}

func PtrBytes(v []byte) *[]byte {
	return &v
}
