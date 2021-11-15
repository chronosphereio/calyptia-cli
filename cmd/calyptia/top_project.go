package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/calyptia/cloud"
	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

var titleStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("62")).
	Foreground(lipgloss.Color("230")).
	Padding(0, 1)

func newCmdTopProject(config *config) *cobra.Command {
	var start, interval time.Duration
	var last uint64
	cmd := &cobra.Command{
		Use:               "project id",
		Short:             "Display metrics from a project",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeProjectIDs,
		// TODO: run an interactive "top" program.
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			var metrics cloud.ProjectMetrics
			var agents []cloud.Agent
			agentMetrics := map[string]cloud.AgentMetrics{}
			var mu sync.Mutex

			g, gctx := errgroup.WithContext(config.ctx)
			g.Go(func() error {
				var err error
				metrics, err = config.cloud.ProjectMetrics(gctx, projectID, start, interval)
				if err != nil {
					return fmt.Errorf("could not fetch metrics: %w", err)
				}

				return nil
			})
			g.Go(func() error {
				var err error
				agents, err = config.cloud.Agents(gctx, projectID, last)
				if err != nil {
					return fmt.Errorf("could not fetch agents: %w", err)
				}

				if len(agents) == 0 {
					return nil
				}

				g1, gctx1 := errgroup.WithContext(gctx)
				for _, a := range agents {
					a := a
					g1.Go(func() error {
						m, err := config.cloud.AgentMetrics(gctx1, a.ID, start, interval)
						if err != nil {
							return fmt.Errorf("could not fetch agent metrics: %w", err)
						}

						mu.Lock()
						agentMetrics[a.ID] = m
						mu.Unlock()

						return nil
					})
				}

				return g1.Wait()
			})
			if err := g.Wait(); err != nil {
				return err
			}

			{
				fmt.Println(titleStyle.Render("Metrics"))

				if len(metrics.Measurements) == 0 {
					fmt.Println("No project metrics to display")
				} else {
					tw := table.NewWriter()
					tw.Style().Options = table.OptionsNoBordersAndSeparators
					if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
						tw.SetAllowedRowLength(w)
					}

					for _, measurementName := range measurementNames(metrics.Measurements) {
						measurement := metrics.Measurements[measurementName]

						for _, pluginName := range pluginNames(measurement.Plugins) {
							// skip internal plugins.
							if strings.HasPrefix(pluginName, "calyptia.") || strings.HasPrefix(pluginName, "fluentbit_metrics.") {
								continue
							}

							plugin := measurement.Plugins[pluginName]
							values := fmtLatestMetrics(plugin.Metrics, interval)
							var value string
							if len(values) == 0 {
								value = "No data"
							} else {
								value = strings.Join(values, ", ")
							}

							tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), value})
						}
					}
					fmt.Println(tw.Render())
				}
			}

			fmt.Println()

			{
				fmt.Println(titleStyle.Render("Agents"))

				if len(agents) == 0 {
					fmt.Println("0 agents")
				} else {
					tw := table.NewWriter()
					tw.Style().Options = table.OptionsNoBordersAndSeparators
					if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
						tw.SetAllowedRowLength(w)
					}

					for _, agent := range agents {
						metrics, ok := agentMetrics[agent.ID]
						if !ok {
							continue
						}

						var mm []string
						for _, measurementName := range agentMeasurementNames(metrics.Measurements) {
							measurement := metrics.Measurements[measurementName]
							values := fmtLatestMetrics(measurement.Totals, interval)
							if len(values) != 0 {
								name := strings.TrimPrefix(measurementName, "fluentbit_")
								name = strings.TrimPrefix(name, "fluentd_")
								mm = append(mm, fmt.Sprintf("%s: %s", name, strings.Join(values, ", ")))
							}
						}

						var value string
						if len(mm) == 0 {
							value = "No data"
						} else {
							value = strings.Join(mm, "\n")
						}

						tw.AppendRow(table.Row{
							agent.Name,
							string(agent.Type) + " " + agent.Version,
							agentStatus(agent.LastMetricsAddedAt),
							value,
						})
					}

					fmt.Println(tw.Render())
				}
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-2, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")

	return cmd
}

func measurementNames(m map[string]cloud.ProjectMeasurement) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func agentMeasurementNames(m map[string]cloud.Measurement) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func pluginNames(m map[string]cloud.Metrics) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func metricNames(m map[string][]cloud.MetricFields) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func fmtFloat64(f float64) string {
	if f > 1 || f < -1 {
		f = math.Round(f)
	}
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

func fmtLatestMetrics(metrics map[string][]cloud.MetricFields, interval time.Duration) []string {
	var values []string

	for _, metricName := range metricNames(metrics) {
		points := metrics[metricName]

		d := len(points)
		if d < 2 {
			continue
		}

		var val *float64
		for i := d - 1; i > 0; i-- {
			curr := points[i].Value
			prev := points[i-1].Value

			if curr == nil || prev == nil {
				continue
			}

			secs := interval.Seconds()
			v := (*curr / secs) - (*prev / secs)
			val = &v
			break
		}

		if val == nil {
			continue
		}

		if strings.Contains(metricName, "dropped_records") {
			values = append(values, fmtFloat64(*val)+"ev/s (dropped)")
			continue
		}

		if strings.Contains(metricName, "retried_records") {
			values = append(values, fmtFloat64(*val)+"ev/s (retried)")
			continue
		}

		if strings.Contains(metricName, "retries_failed") {
			values = append(values, fmtFloat64(*val)+"ev/s (retries failed)")
			continue
		}

		if strings.Contains(metricName, "retries") {
			values = append(values, fmtFloat64(*val)+"ev/s (retries)")
			continue
		}

		if strings.Contains(metricName, "byte") || strings.Contains(metricName, "size") {
			values = append(values, strings.ToLower(bytefmt.ByteSize(uint64(math.Round(*val))))+"/s (bytes)")
			continue
		}

		if strings.Contains(metricName, "record") {
			values = append(values, fmtFloat64(*val)+"ev/s (events)")
			continue
		}

		// TODO: handle "ratio" percentage metrics from fluentd.
		// TODO: handle unknown generic metrics.

		// if strings.Contains(metricName, "ratio") {
		// 	values = append(values, fmtFloat64(*val)+"%")
		// 	continue
		// }

		// values = append(values, fmtFloat64(*val)+"/s")
	}

	return values
}
