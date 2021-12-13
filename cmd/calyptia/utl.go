package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/calyptia/cloud"
	"github.com/hako/durafmt"
)

func fmtAgo(t time.Time) string {
	return durafmt.ParseShort(time.Since(t)).LimitFirstN(1).String()
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
		switch cloud.MeasurementType(measurement) {
		case cloud.FluentbitInputMeasurementType, cloud.FluentdInputMeasurementType:
			rates.InputRecords = rate(points)
		case cloud.FluentbitOutputMeasurementType, cloud.FluentdOutputMeasurementType:
			rates.OutputRecords = rate(points)
		}
		return
	}

	if strings.Contains(metric, "byte") || strings.Contains(metric, "size") {
		switch cloud.MeasurementType(measurement) {
		case cloud.FluentbitInputMeasurementType, cloud.FluentdInputMeasurementType:
			rates.InputBytes = rate(points)
		case cloud.FluentbitOutputMeasurementType, cloud.FluentdOutputMeasurementType:
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

func measurementNames(measurements map[string]cloud.Measurement) []string {
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
