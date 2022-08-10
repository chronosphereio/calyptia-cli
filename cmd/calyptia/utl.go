package main

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/hako/durafmt"

	cloud "github.com/calyptia/api/types"
)

const zeroUUID4 = "00000000-0000-4000-a000-000000000000"

var reUUID4 = regexp.MustCompile("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")

func validUUID(s string) bool {
	return reUUID4.MatchString(s)
}

func fmtTime(t time.Time) string {
	d := time.Since(t)
	if d < time.Second {
		return "Just now"
	}

	return fmtDuration(d)
}

func fmtDuration(d time.Duration) string {
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
			rates.InputRecords = rate(points)
		case "fluentbit_output", "fluentd_output":
			rates.OutputRecords = rate(points)
		}
		return
	}

	if strings.Contains(metric, "byte") || strings.Contains(metric, "size") {
		switch measurement {
		case "fluentbit_input", "fluentd_input":
			rates.InputBytes = rate(points)
		case "fluentbit_output", "fluentd_output":
			rates.OutputBytes = rate(points)
		}
	}
}

func rate(points []cloud.MetricFields) *float64 {
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

func pluginNames(plugins map[string]cloud.Metrics) []string {
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

func projectMeasurementNames(measurements map[string]cloud.ProjectMeasurement) []string {
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

func measurementNames(measurements map[string]cloud.AgentMeasurement) []string {
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

func ptr[T any](p T) *T { return &p }

func filterOutEmptyMetadata(in *json.RawMessage) {
	var o map[string]any
	err := json.Unmarshal(*in, &o)
	if err != nil {
		return
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
	b, err := json.Marshal(o)
	if err != nil {
		return
	}
	c := json.RawMessage(b)
	*in = c
}
