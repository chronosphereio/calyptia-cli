package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"

	cloud "github.com/calyptia/api/types"
	cfg "github.com/calyptia/cli/pkg/config"
	table "github.com/calyptia/go-bubble-table"
)

func newCmdTopAgent(config *cfg.Config) *cobra.Command {
	var start, interval time.Duration

	cmd := &cobra.Command{
		Use:               "agent AGENT",
		Short:             "Display metrics from an agent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.CompleteAgents,
		RunE: func(cmd *cobra.Command, args []string) error {
			agentKey := args[0]
			_, err := tea.NewProgram(initialAgentModel(config.Ctx, config.Cloud, config.ProjectID, agentKey, start, interval), tea.WithAltScreen()).Run()
			return err
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-3, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")

	return cmd
}

func NewAgentModel(ctx context.Context, cloud Client, projectID, agentKey string, metricsStart, metricsInterval time.Duration) AgentModel {
	tbl := table.New([]string{"PLUGIN", "INPUT-BYTES", "INPUT-RECORDS", "OUTPUT-BYTES", "OUTPUT-RECORDS"}, 0, 0)
	return AgentModel{
		projectID:       projectID,
		agentKey:        agentKey,
		metricsStart:    metricsStart,
		metricsInterval: metricsInterval,
		cloud:           cloud,
		ctx:             ctx,
		loading:         true,
		table:           tbl,
	}
}

type AgentModel struct {
	projectID       string
	agentKey        string
	metricsStart    time.Duration
	metricsInterval time.Duration
	cloud           Client
	ctx             context.Context

	cancelFunc  context.CancelFunc
	backEnabled bool
	loading     bool
	err         error
	agentID     string
	agent       cloud.Agent
	tableRows   []table.Row
	table       table.Model
}

func (m *AgentModel) SetData(agent cloud.Agent, metrics cloud.AgentMetrics) {
	m.loading = false
	m.err = nil
	m.agentKey = agent.ID
	m.agentID = agent.ID
	m.agent = agent
	m.tableRows = agentMetricsToTableRows(metrics)
	m.table.SetRows(m.tableRows)
	if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		m.table.SetSize(w, h-1)
	}
}

func (m *AgentModel) SetBackEnabled(v bool) {
	m.backEnabled = v
}

func (m AgentModel) Init() tea.Cmd {
	if m.agentID == "" {
		return m.loadAgentID
	}

	return nil
}

func (m AgentModel) ReloadData() tea.Msg {
	return ReloadAgentDataRequested{}
}

type ReloadAgentDataRequested struct{}

func (m AgentModel) loadAgentID() tea.Msg {
	aa, err := m.cloud.Agents(m.ctx, m.projectID, cloud.AgentsParams{
		Name: &m.agentKey,
	})
	if err != nil {
		return GotAgentError{err}
	}

	if len(aa.Items) != 1 && !validUUID(m.agentKey) {
		if len(aa.Items) != 0 {
			return GotAgentError{fmt.Errorf("ambiguous agent name %q, use ID instead", m.agentKey)}
		}

		return GotAgentError{fmt.Errorf("could not find agent %q", m.agentKey)}
	}

	if len(aa.Items) == 1 {
		return GotAgent{aa.Items[0]}
	}

	return GotAgentID{m.agentKey}
}

type GotAgent struct {
	Agent cloud.Agent
}

type GotAgentID struct {
	AgentID string
}

func (m AgentModel) loadData(ctx context.Context, withAgent, skipError bool) tea.Cmd {
	return func() tea.Msg {
		if !withAgent {
			metrics, err := m.cloud.AgentMetricsV1(ctx, m.agentID, cloud.MetricsParams{
				Start:    m.metricsStart,
				Interval: m.metricsInterval,
			})
			if err != nil {
				// cancelled
				if ctx.Err() != nil {
					return nil
				}

				if skipError {
					return GotAgentError{nil}
				}

				return GotAgentError{err}
			}

			return GotAgentData{
				WithAgent:    withAgent,
				AgentMetrics: metrics,
			}
		}

		var agent cloud.Agent
		var agentMetrics cloud.AgentMetrics
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			var err error
			agent, err = m.cloud.Agent(gctx, m.agentID)
			return err
		})
		g.Go(func() error {
			var err error
			agentMetrics, err = m.cloud.AgentMetricsV1(gctx, m.agentID, cloud.MetricsParams{
				Start:    m.metricsStart,
				Interval: m.metricsInterval,
			})
			return err
		})
		if err := g.Wait(); err != nil {
			// cancelled
			if ctx.Err() != nil {
				return nil
			}

			if skipError {
				return GotAgentError{nil}
			}

			return GotAgentError{err}
		}

		return GotAgentData{
			WithAgent:    withAgent,
			Agent:        agent,
			AgentMetrics: agentMetrics,
		}
	}
}

type GotAgentError struct {
	Err error
}

type GotAgentData struct {
	WithAgent    bool
	Agent        cloud.Agent
	AgentMetrics cloud.AgentMetrics
}

func (m AgentModel) Update(msg tea.Msg) (AgentModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "tab", "backspace":
			if m.backEnabled {
				if m.cancelFunc != nil {
					m.cancelFunc()
				}
				return m, NavigateBackToProject
			}
		}

	case tea.WindowSizeMsg:
		m.table.SetSize(msg.Width, msg.Height-1)
		return m, nil

	case GotAgentID:
		m.agentID = msg.AgentID
		return m, m.loadData(m.ctx, true, false)

	case GotAgentError:
		m.loading = false
		m.err = msg.Err
		if m.err == nil {
			return m, m.ReloadData
		}
		return m, nil

	case GotAgent:
		m.agent = msg.Agent
		m.agentID = msg.Agent.ID
		return m, m.loadData(m.ctx, false, false)

	case ReloadAgentDataRequested:
		var ctx context.Context
		ctx, m.cancelFunc = context.WithCancel(m.ctx)
		return m, tea.Tick(time.Second*5, func(time.Time) tea.Msg {
			return m.loadData(ctx, true, true)()
		})

	case GotAgentData:
		m.loading = false
		m.err = nil
		if msg.WithAgent {
			m.agent = msg.Agent
			m.agentID = msg.Agent.ID
		}
		m.tableRows = agentMetricsToTableRows(msg.AgentMetrics)
		m.table.SetRows(m.tableRows)
		return m, m.ReloadData
	}

	if !m.loading && m.err == nil {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AgentModel) View() string {
	if m.loading {
		return "Loading data... please wait"
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(fmt.Sprintf("Agent %q metrics", m.agent.Name)),
		m.viewMetrics(),
	)
}

func (m AgentModel) viewMetrics() string {
	if len(m.tableRows) == 0 {
		return "No metrics"
	}

	return m.table.View()
}

func agentMetricsToTableRows(metrics cloud.AgentMetrics) []table.Row {
	var rows []table.Row
	for _, measurementName := range measurementNames(metrics.Measurements) {
		measurement := metrics.Measurements[measurementName]
		for _, pluginName := range metricPluginNames(measurement.Plugins) {
			// skip internal metrics.
			if strings.HasPrefix(pluginName, "fluentbit_metrics.") || strings.HasPrefix(pluginName, "calyptia.") {
				continue
			}

			plugin := measurement.Plugins[pluginName]
			row := AgentMetricsRow{PluginName: pluginName}
			for metricName, points := range plugin.Metrics {
				row.Rates.Apply(measurementName, metricName, points)
			}
			rows = append(rows, row)
		}
	}
	return rows
}

type AgentMetricsRow struct {
	PluginName string
	Rates      Rates
}

func (row AgentMetricsRow) Render(w io.Writer, m table.Model, i int) {
	str := fmt.Sprintf("%s\t%s\t%s\t%s\t%s", row.PluginName, ByteCell{row.Rates.InputBytes}, RecordCell{row.Rates.InputRecords}, ByteCell{row.Rates.OutputBytes}, RecordCell{row.Rates.OutputRecords})
	if m.Cursor() == i {
		str = selectedRowStyle.Render(str)
	} else if !row.Rates.OK() {
		str = disabledRowStyle.Render(str)
	}
	fmt.Fprintln(w, str)
}
