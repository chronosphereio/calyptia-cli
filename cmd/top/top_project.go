package top

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"

	"github.com/calyptia/api/client"
	cloud "github.com/calyptia/api/types"
	"github.com/calyptia/cli/cmd/utils"
	cfg "github.com/calyptia/cli/config"
	table "github.com/calyptia/go-bubble-table"
)

func newCmdTopProject(config *cfg.Config) *cobra.Command {
	var start, interval time.Duration
	var last uint
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Display metrics from the current project",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := tea.NewProgram(initialProjectModel(config.Ctx, *config.Cloud, config.ProjectID, start, interval, last), tea.WithAltScreen()).Run()
			return err
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-3, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")
	fs.UintVarP(&last, "last", "l", 0, "Last `N` agents. 0 means no limit")

	return cmd
}

func NewProjectModel(ctx context.Context, cloud client.Client, projectID string, metricsStart, metricsInterval time.Duration, last uint) ProjectModel {
	// TODO: disable project table nivigation.
	projectTable := table.New([]string{"PLUGIN", "INPUT-BYTES", "INPUT-RECORDS", "OUTPUT-BYTES", "OUTPUT-RECORDS"}, 0, 0)
	agentsTable := table.New([]string{"AGENT", "TYPE", "VERSION", "INPUT-BYTES", "INPUT-RECORDS", "OUTPUT-BYTES", "OUTPUT-RECORDS"}, 0, 0)
	return ProjectModel{
		projectID:       projectID,
		metricsStart:    metricsStart,
		metricsInterval: metricsInterval,
		last:            last,
		cloud:           cloud,
		ctx:             ctx,
		loading:         true,
		projectTable:    projectTable,
		agentsTable:     agentsTable,
	}
}

type ProjectModel struct {
	metricsStart    time.Duration
	metricsInterval time.Duration
	last            uint
	cloud           client.Client
	ctx             context.Context

	cancelFunc       context.CancelFunc
	loading          bool
	err              error
	projectID        string
	project          cloud.Project
	projectTableRows []table.Row
	projectTable     table.Model
	agentsTableRows  []table.Row
	agentsTable      table.Model
}

func (m ProjectModel) Init() tea.Cmd {
	return m.loadData(m.ctx, false)
}

func (m ProjectModel) ReloadData() tea.Msg {
	return ReloadProjectDataRequested{}
}

type ReloadProjectDataRequested struct{}

type GotProjectError struct {
	Err error
}

func (m ProjectModel) loadData(ctx context.Context, skipError bool) tea.Cmd {
	return func() tea.Msg {
		var project cloud.Project
		var projectMetrics cloud.ProjectMetrics
		var agents []cloud.Agent
		var agentMetricsByAgentID map[string]cloud.AgentMetrics
		var mu sync.Mutex

		g, gctx := errgroup.WithContext(m.ctx)
		g.Go(func() error {
			var err error
			project, err = m.cloud.Project(gctx, m.projectID)
			return err
		})
		g.Go(func() error {
			var err error
			projectMetrics, err = m.cloud.ProjectMetricsV1(gctx, m.projectID, cloud.MetricsParams{
				Start:    m.metricsStart,
				Interval: m.metricsInterval,
			})
			return err
		})
		g.Go(func() error {
			aa, err := m.cloud.Agents(gctx, m.projectID, cloud.AgentsParams{
				Last: &m.last,
			})
			if err != nil {
				return err
			}

			agents = aa.Items

			if len(agents) != 0 {
				agentMetricsByAgentID = map[string]cloud.AgentMetrics{}
				metricsStart := time.Now().Add(m.metricsStart)
				g2, gctx2 := errgroup.WithContext(gctx)
				for _, agent := range agents {
					agent := agent
					// skip metrics request if agent is offline
					if agent.FirstMetricsAddedAt == nil || agent.FirstMetricsAddedAt.IsZero() || agent.FirstMetricsAddedAt.After(metricsStart) ||
						agent.LastMetricsAddedAt == nil || agent.LastMetricsAddedAt.IsZero() || agent.LastMetricsAddedAt.Before(metricsStart) {
						continue
					}

					g2.Go(func() error {
						agentMetrics, err := m.cloud.AgentMetricsV1(gctx2, agent.ID, cloud.MetricsParams{
							Start:    m.metricsStart,
							Interval: m.metricsInterval,
						})
						if err != nil {
							return err
						}

						mu.Lock()
						agentMetricsByAgentID[agent.ID] = agentMetrics
						mu.Unlock()

						return nil
					})
				}
				return g2.Wait()
			}

			return nil
		})
		if err := g.Wait(); err != nil {
			if ctx.Err() != nil {
				// cancelled
				return nil
			}

			if skipError {
				return GotProjectError{nil}
			}

			return GotProjectError{err}
		}

		return GotProjectData{
			Project:               project,
			ProjectMetrics:        projectMetrics,
			Agents:                agents,
			AgentMetricsByAgentID: agentMetricsByAgentID,
		}
	}
}

type GotProjectData struct {
	Project               cloud.Project
	ProjectMetrics        cloud.ProjectMetrics
	Agents                []cloud.Agent
	AgentMetricsByAgentID map[string]cloud.AgentMetrics
}

func (m *ProjectModel) updateSizes(width, height int) {
	half := int(math.Floor(float64(height-2) / 2))

	if projRemaining := half - len(m.projectTableRows) - 1; projRemaining > 0 {
		m.projectTable.SetSize(width, half-projRemaining)
		m.agentsTable.SetSize(width, half+projRemaining)
	} else {
		m.projectTable.SetSize(width, half)
		m.agentsTable.SetSize(width, half)
	}
}

func (m ProjectModel) Update(msg tea.Msg) (ProjectModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if !m.loading && m.err == nil {
				row, ok := m.agentsTable.SelectedRow().(AgentTotalMetricsRow)
				if ok {
					return m, NavigateToAgent(row.Agent, row.Metrics)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.updateSizes(msg.Width, msg.Height)
		return m, nil
	case ReloadAgentDataRequested:
		var ctx context.Context
		ctx, m.cancelFunc = context.WithCancel(m.ctx)
		return m, tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
			return m.loadData(ctx, true)()
		})
	case GotProjectData:
		m.loading = false
		m.err = nil
		m.project = msg.Project
		m.projectID = msg.Project.ID

		m.projectTableRows = projectMetricsToTableRows(msg.ProjectMetrics)
		m.projectTable.SetRows(m.projectTableRows)

		m.agentsTableRows = agentsWithMetricsToTableRows(msg.Agents, msg.AgentMetricsByAgentID)
		m.agentsTable.SetRows(m.agentsTableRows)

		if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
			m.updateSizes(w, h)
		}

		return m, m.ReloadData
	case GotProjectError:
		m.loading = false
		m.err = msg.Err
		if m.err == nil {
			return m, m.ReloadData
		}
		return m, nil
	}

	if !m.loading && m.err == nil {
		// TODO: scroll one table at a time by using focus.
		var cmds []tea.Cmd
		{
			var cmd tea.Cmd
			m.projectTable, cmd = m.projectTable.Update(msg)
			cmds = append(cmds, cmd)
		}
		{
			var cmd tea.Cmd
			m.agentsTable, cmd = m.agentsTable.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
	}

	return m, nil
}

var titleStyle = lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Bold(true)

func (m ProjectModel) View() string {
	if m.loading {
		return "Loading data... please wait"
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(fmt.Sprintf("Project %q overview", m.project.Name)),
		m.viewOverview(),
		titleStyle.Render("Agents"),
		m.viewAgents(),
	)
}

func (m ProjectModel) viewOverview() string {
	if len(m.projectTableRows) == 0 {
		return "No metrics"
	}

	return m.projectTable.View()
}

func (m ProjectModel) viewAgents() string {
	if len(m.agentsTableRows) == 0 {
		return "No agents"
	}

	return m.agentsTable.View()
}

func projectMetricsToTableRows(metrics cloud.ProjectMetrics) []table.Row {
	var rows []table.Row
	for _, measurementName := range utils.ProjectMeasurementNames(metrics.Measurements) {
		measurement := metrics.Measurements[measurementName]
		for _, pluginName := range utils.MetricPluginNames(measurement.Plugins) {
			// skip internal metrics.
			if strings.HasPrefix(pluginName, "fluentbit_metrics.") || strings.HasPrefix(pluginName, "calyptia.") {
				continue
			}

			plugin := measurement.Plugins[pluginName]
			row := ProjectMetricRow{Plugin: pluginName}
			for metricName, points := range plugin.Metrics {
				row.Rates.Apply(measurementName, metricName, points)
			}
			rows = append(rows, row)
		}
	}
	return rows
}

type ProjectMetricRow struct {
	Plugin string
	Rates  utils.Rates
}

func (row ProjectMetricRow) Render(w io.Writer, m table.Model, i int) {
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", row.Plugin, utils.ByteCell{Value: row.Rates.InputBytes}, utils.RecordCell{Value: row.Rates.InputRecords}, utils.ByteCell{Value: row.Rates.OutputBytes}, utils.RecordCell{Value: row.Rates.OutputRecords})
}

func agentsWithMetricsToTableRows(agents []cloud.Agent, metricsByID map[string]cloud.AgentMetrics) []table.Row {
	var rows []table.Row
	for _, agent := range agents {
		row := AgentTotalMetricsRow{Agent: agent}
		metrics, ok := metricsByID[agent.ID]
		if !ok {
			rows = append(rows, row)
			continue
		}

		row.Metrics = metrics

		for measurementName, measurement := range metrics.Measurements {
			for metricName, points := range measurement.Totals {
				row.Rates.Apply(measurementName, metricName, points)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

type AgentTotalMetricsRow struct {
	Agent   cloud.Agent
	Metrics cloud.AgentMetrics
	Rates   utils.Rates
}

var (
	selectedRowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	disabledRowStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#847A85", Dark: "#979797"})
)

func (row AgentTotalMetricsRow) Render(w io.Writer, m table.Model, i int) {
	str := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s", row.Agent.Name, row.Agent.Type, row.Agent.Version, utils.ByteCell{Value: row.Rates.InputBytes}, utils.RecordCell{Value: row.Rates.InputRecords}, utils.ByteCell{Value: row.Rates.OutputBytes}, utils.RecordCell{Value: row.Rates.OutputRecords})
	if m.Cursor() == i {
		str = selectedRowStyle.Render(str)
	} else if !row.Rates.OK() {
		str = disabledRowStyle.Render(str)
	}
	fmt.Fprintln(w, str)
}
