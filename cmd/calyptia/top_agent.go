package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/calyptia/cloud"
	cloudclient "github.com/calyptia/cloud/client"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hako/durafmt"
	"github.com/muesli/reflow/wordwrap"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func newCmdTopAgent(config *config) *cobra.Command {
	var start, interval time.Duration

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Display metrics from an agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completeAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentKey := args[0]
			initialModel := &Model{
				StartingAgentKey: agentKey,
				AgentModel:       NewAgentModel(config.ctx, config.cloud, agentKey, start, interval),
			}
			err := tea.NewProgram(initialModel, tea.WithAltScreen()).Start()
			if err != nil {
				return fmt.Errorf("could not run program: %w", err)
			}

			return nil
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-2, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")

	return cmd
}

func NewAgentModel(ctx context.Context, cloud *cloudclient.Client, agentKey string, metricsStart, metricsInterval time.Duration) *AgentModel {
	return &AgentModel{
		Ctx:             ctx,
		Cloud:           cloud,
		AgentKey:        agentKey,
		agentID:         agentKey,
		MetricsStart:    metricsStart,
		MetricsInterval: metricsInterval,
		loading:         true,
		Spinner: func() spinner.Model {
			m := spinner.NewModel()
			m.Spinner = spinner.Dot
			return m
		}(),
		MetricList: func() list.Model {
			defaultWidth, defaultHeigth := 36, 17
			if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
				defaultWidth = w - docStyle.GetPaddingLeft() - docStyle.GetPaddingRight()
				_ = h
				// TODO: setup view height.
				// defaultHeigth = h
			}

			metricList := list.NewModel([]list.Item{}, agentMetricsItemDelegate{}, defaultWidth, defaultHeigth)
			metricList.SetShowTitle(false)
			metricList.SetShowStatusBar(false)
			metricList.SetShowFilter(false)
			metricList.Styles.HelpStyle.PaddingLeft(0).PaddingBottom(0)
			// agentList.Styles.PaginationStyle.PaddingLeft(0)
			return metricList
		}(),
	}
}

type AgentModel struct {
	agentID string

	AgentKey        string
	MetricsStart    time.Duration
	MetricsInterval time.Duration

	Ctx   context.Context
	Cloud *cloudclient.Client

	Spinner    spinner.Model
	MetricList list.Model

	loading bool
	err     error

	dataReady    bool
	agent        cloud.Agent
	agentMetrics cloud.AgentMetrics
}

func (m *AgentModel) SetAgentID(agentID string) {
	m.agentID = agentID
}

type FetchAgentDataRequestedMsg struct{}

func (m *AgentModel) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		func() tea.Msg { return FetchAgentDataRequestedMsg{} },
	)
}

type GotAgentDataMsg struct {
	Err          error
	Agent        cloud.Agent
	AgentMetrics cloud.AgentMetrics
}

func (m *AgentModel) fetchAgentData() tea.Msg {
	var agent cloud.Agent
	var agentMetrics cloud.AgentMetrics

	{
		aa, err := fetchAllAgents(m.Cloud, m.Ctx)
		if err != nil {
			return GotAgentDataMsg{Err: fmt.Errorf("could not prefeth agents: %w", err)}
		}

		a, ok := findAgentByName(aa, m.AgentKey)
		if !ok && !validUUID(m.agentID) {
			return GotAgentDataMsg{Err: fmt.Errorf("could not find agent %q", m.AgentKey)}
		}

		if ok {
			m.agentID = a.ID
		}
	}

	g, gctx := errgroup.WithContext(m.Ctx)
	g.Go(func() error {
		var err error
		agent, err = m.Cloud.Agent(gctx, m.agentID)
		if err != nil {
			return fmt.Errorf("could not fetch agent: %w", err)
		}

		return nil
	})
	g.Go(func() error {
		var err error
		agentMetrics, err = m.Cloud.AgentMetrics(gctx, m.agentID, m.MetricsStart, m.MetricsInterval)
		if err != nil {
			return fmt.Errorf("could not fetch agent metrics: %w", err)
		}

		return nil
	})
	if err := g.Wait(); err != nil {
		if m.dataReady {
			// Ignore errors if we already have data.
			// TODO: maybe log it to a file?
			return nil
		}

		return GotAgentDataMsg{Err: err}
	}

	return GotAgentDataMsg{
		Agent:        agent,
		AgentMetrics: agentMetrics,
	}
}

func (m *AgentModel) Update(msg tea.Msg) (*AgentModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.dataReady {
			m.MetricList.SetWidth(msg.Width)
			return m, nil
		}

	case FetchAgentDataRequestedMsg:
		m.loading = true
		m.err = nil
		return m, m.fetchAgentData

	case GotAgentDataMsg:
		if err := msg.Err; err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}

		m.loading = false
		m.err = nil
		m.agent = msg.Agent
		m.agentMetrics = msg.AgentMetrics

		items := []list.Item{}
		for _, measurementName := range agentMeasurementNames(m.agentMetrics.Measurements) {
			item := agentMetricsItem{measurement: measurementName}
			measurement := m.agentMetrics.Measurements[measurementName]
			for _, pluginName := range pluginNames(measurement.Plugins) {
				item.plugin = pluginName
				plugin := measurement.Plugins[pluginName]
				item.values = collectAgentMetricValues(plugin.Metrics, m.MetricsInterval)
			}
			items = append(items, item)
		}

		m.MetricList.SetItems(items)

		m.dataReady = true

		return m, tea.Tick(time.Second*30, func(time.Time) tea.Msg {
			return m.fetchAgentData()
		})
	}

	if m.loading {
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	if m.dataReady {
		var cmd tea.Cmd
		m.MetricList, cmd = m.MetricList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *AgentModel) View() string {
	if m.loading {
		return hintStyle.Render(m.Spinner.View() + " Loading...")
	}

	if err := m.err; err != nil {
		limit := 36 - docStyle.GetPaddingLeft() - docStyle.GetPaddingRight()
		if w, _, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
			limit = w - docStyle.GetPaddingLeft() - docStyle.GetPaddingRight()
		}
		return wordwrap.String(errorStyle.Render(fmt.Sprintf("Error: %v", err)), limit)
	}

	if !m.dataReady {
		return ""
	}

	var doc strings.Builder

	doc.WriteString(titleStyle.Render("Agent: " + m.agent.Name))
	doc.WriteString("\n")
	if len(m.agentMetrics.Measurements) == 0 {
		doc.WriteString(hintStyle.Render("No metrics available.\nLast metric received " + durafmt.ParseShort(time.Since(m.agent.LastMetricsAddedAt)).String() + " ago."))
	} else {
		doc.WriteString("\n")
		doc.WriteString(listHeaderStyle.Render(m.viewAgentMetricsListHeader()) + "\n")
		doc.WriteString(m.MetricList.View())
	}

	return doc.String()
}

type agentMetricsItem struct {
	plugin      string
	measurement string
	values      agentMetricValues
}

type agentMetricValues struct {
	size          string
	events        string
	retries       string
	retriedEvents string
	retriesFailed string
	droppedEvents string
}

func (v agentMetricValues) Empty() bool {
	return v.size == "" && v.events == "" && v.retries == "" && v.retriedEvents == "" && v.retriesFailed == "" && v.droppedEvents == ""
}

func collectAgentMetricValues(metrics map[string][]cloud.MetricFields, interval time.Duration) agentMetricValues {
	var out agentMetricValues

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
			out.droppedEvents = fmtFloat64(*val) + "ev/s"
			continue
		}

		if strings.Contains(metricName, "retried_records") {
			out.retriedEvents = fmtFloat64(*val) + "ev/s"
			continue
		}

		if strings.Contains(metricName, "retries_failed") {
			out.retriesFailed = fmtFloat64(*val) + "ev/s"
			continue
		}

		if strings.Contains(metricName, "retries") {
			out.retries = fmtFloat64(*val) + "ev/s"
			continue
		}

		if strings.Contains(metricName, "byte") || strings.Contains(metricName, "size") {
			out.size = strings.ToLower(bytefmt.ByteSize(uint64(math.Round(*val)))) + "/s"
			continue
		}

		if strings.Contains(metricName, "record") {
			out.events = fmtFloat64(*val) + "ev/s"
			continue
		}

		// TODO: handle "ratio" percentage metrics from fluentd.
		// TODO: handle unknown generic metrics.
	}

	return out
}

func (i agentMetricsItem) FilterValue() string {
	return i.plugin + " " + i.measurement
}

type agentMetricsItemDelegate struct{}

func (d agentMetricsItemDelegate) Height() int                               { return 1 }
func (d agentMetricsItemDelegate) Spacing() int                              { return 0 }
func (d agentMetricsItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d agentMetricsItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(agentMetricsItem)
	if !ok {
		return
	}

	var cells []string
	{
		max := maxAgentMetricsListColumn(m, lipgloss.Width("METRIC"), func(i agentMetricsItem) string { return i.plugin })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.plugin))
	}
	if i.values.size != "" {
		max := maxAgentMetricsListColumn(m, lipgloss.Width("SIZE"), func(i agentMetricsItem) string { return i.values.size })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.size))
	}
	if i.values.events != "" {
		max := maxAgentMetricsListColumn(m, lipgloss.Width("EVENTS"), func(i agentMetricsItem) string { return i.values.events })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.events))
	}
	if i.values.retries != "" {
		max := maxAgentMetricsListColumn(m, lipgloss.Width("RETRIES"), func(i agentMetricsItem) string { return i.values.retries })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.retries))
	}
	if i.values.retriedEvents != "" {
		max := maxAgentMetricsListColumn(m, lipgloss.Width("RETRIED EVENTS"), func(i agentMetricsItem) string { return i.values.retriedEvents })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.retriedEvents))
	}
	if i.values.retriesFailed != "" {
		max := maxAgentMetricsListColumn(m, lipgloss.Width("RETRIES FAILED"), func(i agentMetricsItem) string { return i.values.retriesFailed })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.retriesFailed))
	}
	if i.values.droppedEvents != "" {
		max := maxAgentMetricsListColumn(m, lipgloss.Width("DROPPED EVENTS"), func(i agentMetricsItem) string { return i.values.droppedEvents })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.droppedEvents))
	}

	str := strings.Join(cells, "  ")

	fn := itemStyle.Render
	if i.values.Empty() {
		fn = inactiveItemStyle.Render
	}

	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

func (m *AgentModel) viewAgentMetricsListHeader() string {
	var cells []string
	{
		max := maxAgentMetricsListColumn(m.MetricList, lipgloss.Width("METRIC"), func(i agentMetricsItem) string { return i.plugin })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render("METRIC"))
	}
	{
		max := maxAgentMetricsListColumn(m.MetricList, 0, func(i agentMetricsItem) string { return i.values.size })
		if max != 0 {
			if w := lipgloss.Width("SIZE"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("SIZE"))
		}
	}
	{
		max := maxAgentMetricsListColumn(m.MetricList, 0, func(i agentMetricsItem) string { return i.values.events })
		if max != 0 {
			if w := lipgloss.Width("EVENTS"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("EVENTS"))
		}
	}
	{
		max := maxAgentMetricsListColumn(m.MetricList, 0, func(i agentMetricsItem) string { return i.values.retries })
		if max != 0 {
			if w := lipgloss.Width("RETRIES"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("RETRIES"))
		}
	}
	{
		max := maxAgentMetricsListColumn(m.MetricList, 0, func(i agentMetricsItem) string { return i.values.retriedEvents })
		if max != 0 {
			if w := lipgloss.Width("RETRIED EVENTS"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("RETRIED EVENTS"))
		}
	}
	{
		max := maxAgentMetricsListColumn(m.MetricList, 0, func(i agentMetricsItem) string { return i.values.retriesFailed })
		if max != 0 {
			if w := lipgloss.Width("RETRIES FAILED"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("RETRIES FAILED"))
		}
	}
	{
		max := maxAgentMetricsListColumn(m.MetricList, 0, func(i agentMetricsItem) string { return i.values.droppedEvents })
		if max != 0 {
			if w := lipgloss.Width("DROPPED EVENTS"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("DROPPED EVENTS"))
		}
	}
	return strings.Join(cells, "  ")
}

func maxAgentMetricsListColumn(m list.Model, min int, fn func(agentMetricsItem) string) int {
	max := min
	for _, v := range m.Items() {
		if item, ok := v.(agentMetricsItem); ok {
			if w := lipgloss.Width(fn(item)); w > max {
				max = w
			}
		}
	}
	return max
}
