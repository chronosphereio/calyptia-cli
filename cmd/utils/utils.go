package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/hako/durafmt"

	cloud "github.com/calyptia/api/types"
)

const (
	//nolint: gosec // this is not a secret leak, it's just a format declaration.
	DefaultCoreDockerImage = "ghcr.io/calyptia/core"
)

func FmtTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Second {
		return "Just now"
	}

	return FmtDuration(d)
}

func FmtDuration(d time.Duration) string {
	return durafmt.ParseShort(d).LimitFirstN(1).String()
}

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
			rates.InputRecords = Rate(points)
		case "fluentbit_output", "fluentd_output":
			rates.OutputRecords = Rate(points)
		}
		return
	}

	if strings.Contains(metric, "byte") || strings.Contains(metric, "size") {
		switch measurement {
		case "fluentbit_input", "fluentd_input":
			rates.InputBytes = Rate(points)
		case "fluentbit_output", "fluentd_output":
			rates.OutputBytes = Rate(points)
		}
	}
}

func Rate(points []cloud.MetricFields) *float64 {
	// Only 2 points are required to calc a rate, but the last one is not
	// consistent with the interval. So we actually require 3 points
	// and ignore the last one.
	if len(points) < 3 {
		return nil
	}

	curr := points[len(points)-2]
	prev := points[len(points)-3]

	if curr.Value == nil || prev.Value == nil {
		return nil
	}

	// Rate over a counter should always increase.
	// If it's not, we think of it as a count reset and we ignore it.
	if *curr.Value < *prev.Value {
		return nil
	}

	unit := curr.Time.Sub(prev.Time).Seconds()
	rate := (*curr.Value - *prev.Value) / unit

	return &rate
}

func MetricPluginNames(plugins map[string]cloud.Metrics) []string {
	if len(plugins) == 0 {
		return nil
	}

	names := make([]string, 0, len(plugins))
	for k := range plugins {
		names = append(names, k)
	}
	sort.Stable(sort.StringSlice(names))
	return names
}

func ProjectMeasurementNames(measurements map[string]cloud.ProjectMeasurement) []string {
	if len(measurements) == 0 {
		return nil
	}

	names := make([]string, 0, len(measurements))
	for k := range measurements {
		names = append(names, k)
	}
	sort.Stable(sort.StringSlice(names))
	return names
}

func MeasurementNames(measurements map[string]cloud.AgentMeasurement) []string {
	if len(measurements) == 0 {
		return nil
	}

	names := make([]string, 0, len(measurements))
	for k := range measurements {
		names = append(names, k)
	}
	sort.Stable(sort.StringSlice(names))
	return names
}

func FilterOutEmptyMetadata(metadata cloud.CoreInstanceMetadata) ([]byte, error) {
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	var o map[string]any
	err = json.Unmarshal(b, &o)
	if err != nil {
		return nil, err
	}
	for k, v := range o {
		switch v.(type) {
		case float64, int:
			v, ok := v.(float64)
			if !ok {
				continue
			}
			if v <= 0 {
				delete(o, k)
			}
		default:
			v, ok := v.(string)
			if !ok {
				continue
			}
			if v == "" {
				delete(o, k)
			}
		}
	}

	return json.Marshal(o)
}

func ReadConfirm(r io.Reader) (bool, error) {
	var answer string
	_, err := fmt.Fscanln(r, &answer)
	if err != nil && err.Error() == "unexpected newline" {
		err = nil
	}

	if err != nil {
		return false, fmt.Errorf("could not to read answer: %v", err)
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
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
