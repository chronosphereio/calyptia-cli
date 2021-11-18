package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/calyptia/cloud"
	cloudclient "github.com/calyptia/cloud/client"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
		Use:               "project PROJECT",
		Short:             "Display metrics from a project",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeProjects,
		// TODO: run an interactive "top" program.
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := args[0]
			initialModel := NewModel(
				WithProject(NewProjectModel(config.ctx, config.cloud, projectKey, start, interval, last)),
			)
			p := tea.NewProgram(initialModel)
			p.EnterAltScreen()

			err := p.Start()
			if err != nil {
				return fmt.Errorf("could not run program: %w", err)
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

var (
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
)

func NewProjectModel(ctx context.Context, cloud *cloudclient.Client, projectKey string, metricsStart, metricsInterval time.Duration, lastAgents uint64) *ProjectModel {
	return &ProjectModel{
		Ctx:             ctx,
		Cloud:           cloud,
		ProjectKey:      projectKey,
		projectID:       projectKey,
		MetricsStart:    metricsStart,
		MetricsInterval: metricsInterval,
		LastAgents:      lastAgents,
		loading:         true,
	}
}

type ProjectModel struct {
	Ctx             context.Context
	Cloud           *cloudclient.Client
	ProjectKey      string
	projectID       string
	MetricsStart    time.Duration
	MetricsInterval time.Duration
	LastAgents      uint64

	loading bool
	err     error

	agentList      list.Model
	gotData        bool
	projectMetrics cloud.ProjectMetrics
	agents         []cloud.Agent
	agentMetrics   map[string]cloud.AgentMetrics
}

type FetchProjectDataRequestedMsg struct{}

func (m *ProjectModel) Init() tea.Cmd {
	return func() tea.Msg { return FetchProjectDataRequestedMsg{} }
}

type GotProjectDataMsg struct {
	Err            error
	ProjectMetrics cloud.ProjectMetrics
	Agents         []cloud.Agent
	AgentMetrics   map[string]cloud.AgentMetrics
}

func (m *ProjectModel) fetchProjectData() tea.Msg {
	var projectMetrics cloud.ProjectMetrics
	var agents []cloud.Agent
	agentMetrics := map[string]cloud.AgentMetrics{}
	var mu sync.Mutex

	if !validUUID(m.projectID) {
		pp, err := m.Cloud.Projects(m.Ctx, 0)
		if err != nil {
			return GotProjectDataMsg{Err: fmt.Errorf("could not prefeth projects: %w", err)}
		}

		p, ok := findProjectByName(pp, m.ProjectKey)
		if !ok {
			return GotProjectDataMsg{Err: fmt.Errorf("could not find project %q", m.ProjectKey)}
		}

		m.projectID = p.ID
	}

	g, gctx := errgroup.WithContext(m.Ctx)
	g.Go(func() error {
		var err error
		projectMetrics, err = m.Cloud.ProjectMetrics(gctx, m.projectID, m.MetricsStart, m.MetricsInterval)
		if err != nil {
			return fmt.Errorf("could not fetch metrics: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error
		agents, err = m.Cloud.Agents(gctx, m.projectID, m.LastAgents)
		if err != nil {
			return fmt.Errorf("could not fetch agents: %w", err)
		}

		if len(agents) == 0 {
			return nil
		}

		g1, gctx1 := errgroup.WithContext(gctx)
		for _, a := range agents {
			// Avoid metrics request if we know last metric was added before `start`.
			inactive := a.LastMetricsAddedAt.IsZero() || a.LastMetricsAddedAt.Before(time.Now().Add(m.MetricsStart))
			if inactive {
				continue
			}

			a := a
			g1.Go(func() error {
				m, err := m.Cloud.AgentMetrics(gctx1, a.ID, m.MetricsStart, m.MetricsInterval)
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
		return GotProjectDataMsg{Err: err}
	}

	return GotProjectDataMsg{
		ProjectMetrics: projectMetrics,
		Agents:         agents,
		AgentMetrics:   agentMetrics,
	}
}

func (m *ProjectModel) Update(msg tea.Msg) (*ProjectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.gotData {
			m.agentList.SetWidth(msg.Width)
		}
		return m, nil
	case FetchProjectDataRequestedMsg:
		m.loading = true
		return m, m.fetchProjectData
	case GotProjectDataMsg:
		m.loading = false
		m.gotData = true

		if err := msg.Err; err != nil {
			m.err = err
			return m, nil
		}

		m.projectMetrics = msg.ProjectMetrics
		m.agents = msg.Agents
		m.agentMetrics = msg.AgentMetrics

		items := make([]list.Item, len(m.agents))
		for i, a := range m.agents {
			item := agentListItem{
				agent: a,
			}
			if metrics, ok := m.agentMetrics[a.ID]; ok {
				for _, measurementName := range agentMeasurementNames(metrics.Measurements) {
					measurement := metrics.Measurements[measurementName]
					values := fmtLatestMetrics(measurement.Totals, m.MetricsInterval)
					if len(values) != 0 {
						value := strings.Join(values, ", ")
						switch cloud.MeasurementType(measurementName) {
						case cloud.FluentbitInputMeasurementType, cloud.FluentdInputMeasurementType:
							item.values.input = value
						case cloud.FluentbitOutputMeasurementType, cloud.FluentdOutputMeasurementType:
							item.values.output = value
						case cloud.FluentbitFilterMeasurementType, cloud.FluentdFilterMeasurementType:
							item.values.filter = value
						case cloud.FluentbitStorageMeasurementType, cloud.FluentdStorageMeasurementType:
							item.values.storage = value
						case cloud.FluentdMultiOutputMeasurementType:
							item.values.multiOutput = value
						case cloud.FluentdBareOutputMeasurementType:
							item.values.bareOutput = value
						}
					}
				}
			}
			items[i] = item
		}

		defaultWidth, defaultHeigth := 36, 17
		// if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		// 	defaultWidth = w
		// 	defaultHeigth = h
		// }

		m.agentList = list.NewModel(items, itemDelegate{}, defaultWidth, defaultHeigth)
		m.agentList.Title = "Agents"

		return m, nil
	}

	if m.gotData {
		var cmd tea.Cmd
		m.agentList, cmd = m.agentList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ProjectModel) View() string {
	if m.loading {
		return "Loading..."
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if !m.gotData {
		return ""
	}

	var doc strings.Builder

	{
		doc.WriteString(titleStyle.Render("Metrics") + "\n")

		if len(m.projectMetrics.Measurements) == 0 {
			doc.WriteString("No project metrics to display\n")
		} else {
			tw := table.NewWriter()
			tw.Style().Options = table.OptionsNoBordersAndSeparators
			if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				tw.SetAllowedRowLength(w)
			}

			for _, measurementName := range measurementNames(m.projectMetrics.Measurements) {
				measurement := m.projectMetrics.Measurements[measurementName]

				for _, pluginName := range pluginNames(measurement.Plugins) {
					// skip internal plugins.
					if strings.HasPrefix(pluginName, "calyptia.") || strings.HasPrefix(pluginName, "fluentbit_metrics.") {
						continue
					}

					plugin := measurement.Plugins[pluginName]
					values := fmtLatestMetrics(plugin.Metrics, m.MetricsInterval)
					var value string
					if len(values) == 0 {
						value = "No data"
					} else {
						value = strings.Join(values, ", ")
					}

					tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), value})
				}
			}
			doc.WriteString(tw.Render() + "\n")
		}
	}

	doc.WriteString("\n")
	doc.WriteString(m.agentList.View())

	return doc.String()
}

type agentListItem struct {
	agent  cloud.Agent
	values agentMeasurementValues
}

type agentMeasurementValues struct {
	input       string
	output      string
	filter      string
	storage     string
	multiOutput string
	bareOutput  string
}

func (i agentListItem) FilterValue() string {
	return i.agent.Name + " " + i.agent.ID + " " + string(i.agent.Type)
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(agentListItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s | %s | %s", i.agent.Name, i.agent.Type, i.agent.Version)
	var values []string
	if i.values.input != "" {
		max := maxAgentListValue(m, func(item agentListItem) string { return item.values.input })
		values = append(values, withWhitespace(i.values.input, max))
	}
	if i.values.output != "" {
		max := maxAgentListValue(m, func(item agentListItem) string { return item.values.output })
		values = append(values, withWhitespace(i.values.output, max))
	}
	if i.values.filter != "" {
		max := maxAgentListValue(m, func(item agentListItem) string { return item.values.filter })
		values = append(values, withWhitespace(i.values.filter, max))
	}
	if i.values.storage != "" {
		max := maxAgentListValue(m, func(item agentListItem) string { return item.values.storage })
		values = append(values, withWhitespace(i.values.storage, max))
	}
	if i.values.multiOutput != "" {
		max := maxAgentListValue(m, func(item agentListItem) string { return item.values.multiOutput })
		values = append(values, withWhitespace(i.values.multiOutput, max))
	}
	if i.values.bareOutput != "" {
		max := maxAgentListValue(m, func(item agentListItem) string { return item.values.bareOutput })
		values = append(values, withWhitespace(i.values.bareOutput, max))
	}
	if len(values) != 0 {
		str += " | " + strings.Join(values, " | ")
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

func maxAgentListValue(m list.Model, fn func(agentListItem) string) int {
	var max int
	for _, v := range m.Items() {
		if item, ok := v.(agentListItem); ok {
			if w := lipgloss.Width(fn(item)); w > max {
				max = w
			}
		}
	}
	return max
}

func withWhitespace(s string, max int) string {
	w := lipgloss.Width(s)
	repeat := max - w
	if repeat < 0 {
		repeat = 0
	}
	spaces := strings.Repeat(" ", repeat)
	return spaces + s
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

			if *curr < *prev {
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
