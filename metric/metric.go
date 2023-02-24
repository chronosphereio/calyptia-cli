package metric

import (
	"sort"

	"github.com/calyptia/api/types"
)

func Rate(points []types.MetricFields) *float64 {
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

func MetricPluginNames(plugins map[string]types.Metrics) []string {
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

func ProjectMeasurementNames(measurements map[string]types.ProjectMeasurement) []string {
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

func MeasurementNames(measurements map[string]types.AgentMeasurement) []string {
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
