package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/calyptia/cloud"
	cloudclient "github.com/calyptia/cloud/client"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/muesli/reflow/wordwrap"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

var (
	docStyle           = lipgloss.NewStyle().Padding(1)
	hintStyle          = lipgloss.NewStyle().Padding(1, 0, 0, 2).Foreground(lipgloss.AdaptiveColor{Light: "#847A85", Dark: "#979797"})
	titleStyle         = lipgloss.NewStyle().Padding(0, 1).MarginLeft(2).Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Bold(true)
	errorStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF1744"))
	overviewTableStyle = lipgloss.NewStyle().Padding(1)
	itemStyle          = lipgloss.NewStyle().PaddingLeft(2)
	inactiveItemStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.AdaptiveColor{Light: "#847A85", Dark: "#979797"})
	selectedItemStyle  = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("170"))
	listHeaderStyle    = lipgloss.NewStyle().PaddingLeft(2).Bold(true)
)

func newCmdTopProject(config *config) *cobra.Command {
	var start, interval time.Duration
	var last uint64
	cmd := &cobra.Command{
		Use:               "project [PROJECT]",
		Short:             "Display metrics from a project",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: config.completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey := config.defaultProject
			if len(args) > 0 {
				projectKey = args[0]
			}
			if projectKey == "" {
				return errors.New("project required")
			}

			initialModel := &Model{
				StartingProjectKey: projectKey,
				ProjectModel:       NewProjectModel(config.ctx, config.cloud, projectKey, start, interval, last),
				AgentModel:         NewAgentModel(config.ctx, config.cloud, projectKey, start, interval),
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
	fs.Uint64VarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")

	return cmd
}

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
		Spinner: func() spinner.Model {
			m := spinner.NewModel()
			m.Spinner = spinner.Dot
			return m
		}(),
		AgentList: func() list.Model {
			defaultWidth, defaultHeigth := 36, 17
			if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
				defaultWidth = w - docStyle.GetPaddingLeft() - docStyle.GetPaddingRight()
				_ = h
				// TODO: setup view height.
				// defaultHeigth = h
			}

			agentList := list.NewModel([]list.Item{}, agentItemDelegate{}, defaultWidth, defaultHeigth)
			agentList.SetShowTitle(false)
			agentList.SetShowStatusBar(false)
			agentList.SetShowFilter(false)
			agentList.Styles.HelpStyle.PaddingLeft(0).PaddingBottom(0)
			// agentList.Styles.PaginationStyle.PaddingLeft(0)
			return agentList
		}(),
	}
}

type ProjectModel struct {
	projectID string

	ProjectKey string
	Ctx        context.Context
	Cloud      *cloudclient.Client

	MetricsStart    time.Duration
	MetricsInterval time.Duration
	LastAgents      uint64

	AgentList list.Model
	Spinner   spinner.Model

	loading bool
	err     error

	dataReady      bool
	projectMetrics cloud.ProjectMetrics
	agents         []cloud.Agent
	agentsMetrics  map[string]cloud.AgentMetrics
}

func (m *ProjectModel) SetProjectID(projectID string) {
	m.projectID = projectID
}

type FetchProjectDataRequestedMsg struct{}

func (m *ProjectModel) Init() tea.Cmd {
	return tea.Batch(
		spinner.Tick,
		func() tea.Msg { return FetchProjectDataRequestedMsg{} },
	)
}

type GotProjectDataMsg struct {
	Err            error
	ProjectMetrics cloud.ProjectMetrics
	Agents         []cloud.Agent
	AgentsMetrics  map[string]cloud.AgentMetrics
}

func (m *ProjectModel) fetchProjectData() tea.Msg {
	var projectMetrics cloud.ProjectMetrics
	var agents []cloud.Agent
	agentMetrics := map[string]cloud.AgentMetrics{}
	var mu sync.Mutex

	{
		pp, err := m.Cloud.Projects(m.Ctx)
		if err != nil {
			return GotProjectDataMsg{Err: fmt.Errorf("could not prefeth projects: %w", err)}
		}

		p, ok := findProjectByName(pp, m.ProjectKey)
		if !ok && !validUUID(m.projectID) {
			return GotProjectDataMsg{Err: fmt.Errorf("could not find project %q", m.ProjectKey)}
		}

		if ok {
			m.projectID = p.ID
		}
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
		agents, err = m.Cloud.Agents(gctx, m.projectID, cloud.LastAgents(m.LastAgents))
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
		if m.dataReady {
			// Ignore errors if we already have data.
			// TODO: maybe log it to a file?
			return nil
		}

		return GotProjectDataMsg{Err: err}
	}

	return GotProjectDataMsg{
		ProjectMetrics: projectMetrics,
		Agents:         agents,
		AgentsMetrics:  agentMetrics,
	}
}

func (m *ProjectModel) Update(msg tea.Msg) (*ProjectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter && m.dataReady {
			item, ok := m.AgentList.SelectedItem().(agentListItem)
			if ok {
				return m, GoToAgentView(item.agent.ID)
			}
		}

	case tea.WindowSizeMsg:
		if m.dataReady {
			m.AgentList.SetWidth(msg.Width)
			return m, nil
		}

	case FetchProjectDataRequestedMsg:
		m.loading = true
		m.err = nil
		return m, m.fetchProjectData

	case GotProjectDataMsg:
		if err := msg.Err; err != nil {
			m.loading = false
			m.err = err
			return m, nil
		}

		m.loading = false
		m.err = nil
		m.projectMetrics = msg.ProjectMetrics
		m.agents = msg.Agents
		m.agentsMetrics = msg.AgentsMetrics

		items := make([]list.Item, len(m.agents))
		for i, a := range m.agents {
			item := agentListItem{agent: a}
			if metrics, ok := m.agentsMetrics[a.ID]; ok {
				item.values = makeAgentMeasurementValues(metrics)
			}
			items[i] = item
		}

		m.AgentList.SetItems(items)
		m.dataReady = true

		return m, tea.Tick(time.Second*30, func(time.Time) tea.Msg {
			return m.fetchProjectData()
		})
	}

	if m.loading {
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	}

	if m.dataReady {
		var cmd tea.Cmd
		m.AgentList, cmd = m.AgentList.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *ProjectModel) View() string {
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

	{
		doc.WriteString(titleStyle.Render("Overview") + "\n")

		if len(m.projectMetrics.Measurements) == 0 {
			doc.WriteString(hintStyle.Render("No overview metrics to display") + "\n")
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
					values := fmtLatestMetrics(plugin.Metrics)
					var value string
					if len(values) == 0 {
						value = "No data"
					} else {
						value = strings.Join(values, ", ")
					}

					tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), value})
				}
			}
			doc.WriteString(overviewTableStyle.Render(tw.Render()) + "\n")
		}
	}

	doc.WriteString(listHeaderStyle.Render(m.viewAgentListHeader()) + "\n")
	doc.WriteString(m.AgentList.View())

	return doc.String()
}

func (m *ProjectModel) viewAgentListHeader() string {
	var cells []string
	{
		max := maxAgentListColumn(m.AgentList, lipgloss.Width("AGENT"), func(item agentListItem) string { return item.agent.Name })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render("AGENT"))
	}
	{
		max := maxAgentListColumn(m.AgentList, lipgloss.Width("TYPE"), func(item agentListItem) string { return string(item.agent.Type) })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render("TYPE"))
	}
	{
		max := maxAgentListColumn(m.AgentList, lipgloss.Width("VERSION"), func(item agentListItem) string { return item.agent.Version })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render("VERSION"))
	}
	{
		max := maxAgentListColumn(m.AgentList, 0, func(item agentListItem) string { return item.values.input })
		if max != 0 {
			if w := lipgloss.Width("INPUT"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("INPUT"))
		}
	}
	{
		max := maxAgentListColumn(m.AgentList, 0, func(item agentListItem) string { return item.values.output })
		if max != 0 {
			if w := lipgloss.Width("OUTPUT"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("OUTPUT"))
		}
	}
	{
		max := maxAgentListColumn(m.AgentList, 0, func(item agentListItem) string { return item.values.filter })
		if max != 0 {
			if w := lipgloss.Width("FILTER"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("FILTER"))
		}
	}
	{
		max := maxAgentListColumn(m.AgentList, 0, func(item agentListItem) string { return item.values.storage })
		if max != 0 {
			if w := lipgloss.Width("STORAGE"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("STORAGE"))
		}
	}
	{
		max := maxAgentListColumn(m.AgentList, 0, func(item agentListItem) string { return item.values.multiOutput })
		if max != 0 {
			if w := lipgloss.Width("MULTI OUTPUT"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("MULTI OUTPUT"))
		}
	}
	{
		max := maxAgentListColumn(m.AgentList, 0, func(item agentListItem) string { return item.values.bareOutput })
		if max != 0 {
			if w := lipgloss.Width("BARE OUTPUT"); w > max {
				max = w
			}
			cells = append(cells, lipgloss.NewStyle().Width(max).Render("BARE OUTPUT"))
		}
	}
	return strings.Join(cells, "  ")
}

func makeAgentMeasurementValues(metrics cloud.AgentMetrics) agentMeasurementValues {
	var out agentMeasurementValues
	for _, measurementName := range agentMeasurementNames(metrics.Measurements) {
		measurement := metrics.Measurements[measurementName]
		values := fmtLatestMetrics(measurement.Totals)
		if len(values) != 0 {
			value := strings.Join(values, ", ")
			switch cloud.MeasurementType(measurementName) {
			case cloud.FluentbitInputMeasurementType, cloud.FluentdInputMeasurementType:
				out.input = value
			case cloud.FluentbitOutputMeasurementType, cloud.FluentdOutputMeasurementType:
				out.output = value
			case cloud.FluentbitFilterMeasurementType, cloud.FluentdFilterMeasurementType:
				out.filter = value
			case cloud.FluentbitStorageMeasurementType, cloud.FluentdStorageMeasurementType:
				out.storage = value
			case cloud.FluentdMultiOutputMeasurementType:
				out.multiOutput = value
			case cloud.FluentdBareOutputMeasurementType:
				out.bareOutput = value
			}
		}
	}
	return out
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

type agentItemDelegate struct{}

func (d agentItemDelegate) Height() int                               { return 1 }
func (d agentItemDelegate) Spacing() int                              { return 0 }
func (d agentItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d agentItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(agentListItem)
	if !ok {
		return
	}

	var cells []string
	{
		max := maxAgentListColumn(m, lipgloss.Width("AGENT"), func(item agentListItem) string { return item.agent.Name })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.agent.Name))
	}
	{
		max := maxAgentListColumn(m, lipgloss.Width("TYPE"), func(item agentListItem) string { return string(item.agent.Type) })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(string(i.agent.Type)))
	}
	{
		max := maxAgentListColumn(m, lipgloss.Width("VERSION"), func(item agentListItem) string { return item.agent.Version })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.agent.Version))
	}
	if i.values.input != "" {
		max := maxAgentListColumn(m, lipgloss.Width("INPUT"), func(item agentListItem) string { return item.values.input })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.input))
	}
	if i.values.output != "" {
		max := maxAgentListColumn(m, lipgloss.Width("OUTPUT"), func(item agentListItem) string { return item.values.output })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.output))
	}
	if i.values.filter != "" {
		max := maxAgentListColumn(m, lipgloss.Width("FILTER"), func(item agentListItem) string { return item.values.filter })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.filter))
	}
	if i.values.storage != "" {
		max := maxAgentListColumn(m, lipgloss.Width("STORAGE"), func(item agentListItem) string { return item.values.storage })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.storage))
	}
	if i.values.multiOutput != "" {
		max := maxAgentListColumn(m, lipgloss.Width("MULTI OUTPUT"), func(item agentListItem) string { return item.values.multiOutput })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.multiOutput))
	}
	if i.values.bareOutput != "" {
		max := maxAgentListColumn(m, lipgloss.Width("BARE OUTPUT"), func(item agentListItem) string { return item.values.bareOutput })
		cells = append(cells, lipgloss.NewStyle().Width(max).Render(i.values.bareOutput))
	}

	str := strings.Join(cells, "  ")

	fn := itemStyle.Render
	if i.values.input == "" && i.values.output == "" && i.values.filter == "" && i.values.storage == "" && i.values.multiOutput == "" && i.values.bareOutput == "" {
		fn = inactiveItemStyle.Render
	}

	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

func maxAgentListColumn(m list.Model, min int, fn func(agentListItem) string) int {
	max := min
	for _, v := range m.Items() {
		if item, ok := v.(agentListItem); ok {
			if w := lipgloss.Width(fn(item)); w > max {
				max = w
			}
		}
	}
	return max
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

func fmtLatestMetrics(metrics map[string][]cloud.MetricFields) []string {
	got := collectAgentMetricValues(metrics)
	var values []string
	if got.size != "" {
		values = append(values, fmt.Sprintf("size: %s", got.size))
	}
	if got.events != "" {
		values = append(values, fmt.Sprintf("events: %s", got.events))
	}
	if got.retries != "" {
		values = append(values, fmt.Sprintf("retries: %s", got.retries))
	}
	if got.retriedEvents != "" {
		values = append(values, fmt.Sprintf("retried events: %s", got.retriedEvents))
	}
	if got.retriesFailed != "" {
		values = append(values, fmt.Sprintf("retries failed: %s", got.retriesFailed))
	}
	if got.droppedEvents != "" {
		values = append(values, fmt.Sprintf("dropped events: %s", got.droppedEvents))
	}
	return values
}
