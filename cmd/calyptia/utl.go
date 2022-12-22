package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	text_template "text/template"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/Masterminds/sprig/v3"
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

func metricPluginNames(plugins map[string]cloud.Metrics) []string {
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

func filterOutEmptyMetadata(metadata cloud.CoreInstanceMetadata) ([]byte, error) {
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

func readConfirm(r io.Reader) (bool, error) {
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

func applyGoTemplate(w io.Writer, outputFormat, goTemplate string, data any) error {
	if goTemplate == "" {
		parts := strings.SplitN(outputFormat, "=", 2)
		if len(parts) != 2 {
			return nil
		}

		goTemplate = trimQuotes(parts[1])

		if goTemplate == "" {
			return nil
		}
	}

	goTemplate = strings.TrimSpace(goTemplate)

	if strings.HasPrefix(outputFormat, "go-template-file") {
		b, err := os.ReadFile(goTemplate)
		if err != nil {
			return fmt.Errorf("reading go-template-file: %w", err)
		}

		goTemplate = string(bytes.TrimSpace(b))
	}

	tmpl, err := text_template.New("").Funcs(sprig.FuncMap()).Parse(goTemplate + "\n")
	if err != nil {
		return fmt.Errorf("parsing go-template: %w", err)
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		return fmt.Errorf("rendering go-template: %w", err)
	}

	return nil
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'' || c == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func uniqueSlice[S ~[]E, E comparable](s S) S {
	m := map[E]struct{}{}

	var out S
	for _, item := range s {
		if _, ok := m[item]; !ok {
			out = append(out, item)
		}
	}
	return out
}

func renderCreatedTable(w io.Writer, createdID string, createdAt time.Time) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "ID\tCREATED-AT")
	_, err := fmt.Fprintf(tw, "%s\t%s\n", createdID, createdAt.Local().Format(time.RFC822))
	if err != nil {
		return err
	}

	return tw.Flush()
}

func renderUpdatedTable(w io.Writer, updatedAt time.Time) error {
	tw := tabwriter.NewWriter(w, 0, 4, 1, ' ', 0)
	fmt.Fprintln(tw, "UPDATED-AT")
	_, err := fmt.Fprintln(tw, updatedAt.Local().Format(time.RFC822))
	if err != nil {
		return err
	}

	return tw.Flush()
}
