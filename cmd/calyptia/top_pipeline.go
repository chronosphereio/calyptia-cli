package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/calyptia/cloud"
	"github.com/calyptia/cloud-cli/cmd/calyptia/bubles/table"
	cloudclient "github.com/calyptia/cloud/client"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
)

func newCmdTopPipeline(config *config) *cobra.Command {
	var start, interval time.Duration

	cmd := &cobra.Command{
		Use:               "pipeline PIPELINE",
		Short:             "Display metrics from a pipeline",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: config.completePipelines,
		RunE: func(cmd *cobra.Command, args []string) error {
			pipelineKey := args[0]
			return tea.NewProgram(initialPipelineModel(config.ctx, config.cloud, pipelineKey, start, interval), tea.WithAltScreen()).Start()
		},
	}

	fs := cmd.Flags()
	fs.DurationVar(&start, "start", time.Minute*-3, "Start time range")
	fs.DurationVar(&interval, "interval", time.Minute, "Interval rate")

	return cmd
}

func NewPipelineModel(ctx context.Context, cloud *cloudclient.Client, pipelineKey string, metricsStart, metricsInterval time.Duration) PipelineModel {
	tbl := table.NewModel([]string{"PLUGIN", "INPUT-BYTES", "INPUT-RECORDS", "OUTPUT-BYTES", "OUTPUT-RECORDS"})
	tbl.SetNavEnabled(true)
	return PipelineModel{
		pipelineKey:     pipelineKey,
		metricsStart:    metricsStart,
		metricsInterval: metricsInterval,
		cloud:           cloud,
		ctx:             ctx,
		loading:         true,
		table:           tbl,
	}
}

type PipelineModel struct {
	pipelineKey     string
	metricsStart    time.Duration
	metricsInterval time.Duration
	cloud           *cloudclient.Client
	ctx             context.Context

	cancelFunc  context.CancelFunc
	backEnabled bool
	loading     bool
	err         error
	pipelineID  string
	pipeline    cloud.AggregatorPipeline
	tableRows   []table.Row
	table       table.Model
}

func (m *PipelineModel) SetData(pipeline cloud.AggregatorPipeline, metrics cloud.AgentMetrics) {
	m.loading = false
	m.err = nil
	m.pipelineKey = pipeline.ID
	m.pipelineID = pipeline.ID
	m.pipeline = pipeline
	m.tableRows = agentMetricsToTableRows(metrics)
	m.table.SetRows(m.tableRows)
	if w, h, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		m.table.SetSize(w, h-1)
	}
}

func (m *PipelineModel) SetBackEnabled(v bool) {
	m.backEnabled = v
}

func (m PipelineModel) Init() tea.Cmd {
	if m.pipelineID == "" {
		return m.loadPipelineID
	}

	return nil
}

func (m PipelineModel) ReloadData() tea.Msg {
	return ReloadPipelineDataRequested{}
}

type ReloadPipelineDataRequested struct{}

func (m PipelineModel) loadPipelineID() tea.Msg {
	pp, err := m.cloud.Projects(m.ctx)
	if err != nil {
		return GotPipelineError{err}
	}

	var founds []cloud.AggregatorPipeline
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(m.ctx)
	for _, p := range pp {
		p := p
		g.Go(func() error {
			aa, err := m.cloud.ProjectPipelines(gctx, p.ID, cloud.PipelinesWithName(m.pipelineKey))
			if err != nil {
				return err
			}

			if len(aa) == 1 {
				mu.Lock()
				founds = append(founds, aa[0])
				mu.Unlock()
			}

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return GotPipelineError{err}
	}

	if len(founds) != 1 && !validUUID(m.pipelineKey) {
		if len(founds) != 0 {
			return GotPipelineError{fmt.Errorf("ambiguous pipeline name %q, use ID instead", m.pipelineKey)}
		}

		return GotPipelineError{fmt.Errorf("could not find pipeline %q", m.pipelineKey)}
	}

	if len(founds) == 1 {
		return GotPipeline{founds[0]}
	}

	return GotPipelineID{m.pipelineKey}
}

type GotPipeline struct {
	Pipeline cloud.AggregatorPipeline
}

type GotPipelineID struct {
	PipelineID string
}

func (m PipelineModel) loadData(ctx context.Context, withPipeline, skipError bool) tea.Cmd {
	return func() tea.Msg {
		if !withPipeline {
			metrics, err := m.cloud.PipelineMetrics(ctx, m.pipelineID, m.metricsStart, m.metricsInterval)
			if err != nil {
				// cancelled
				if ctx.Err() != nil {
					return nil
				}

				if skipError {
					return GotPipelineError{nil}
				}

				return GotPipelineError{err}
			}

			return GotPipelineData{
				WithPipeline:    withPipeline,
				PipelineMetrics: metrics,
			}
		}

		var pipeline cloud.AggregatorPipeline
		var pipelineMetrics cloud.AgentMetrics
		g, gctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			var err error
			pipeline, err = m.cloud.AggregatorPipeline(gctx, m.pipelineID)
			return err
		})
		g.Go(func() error {
			var err error
			pipelineMetrics, err = m.cloud.PipelineMetrics(gctx, m.pipelineID, m.metricsStart, m.metricsInterval)
			return err
		})
		if err := g.Wait(); err != nil {
			// cancelled
			if ctx.Err() != nil {
				return nil
			}

			if skipError {
				return GotPipelineError{nil}
			}

			return GotPipelineError{err}
		}

		return GotPipelineData{
			WithPipeline:    withPipeline,
			Pipeline:        pipeline,
			PipelineMetrics: pipelineMetrics,
		}
	}
}

type GotPipelineError struct {
	Err error
}

type GotPipelineData struct {
	WithPipeline    bool
	Pipeline        cloud.AggregatorPipeline
	PipelineMetrics cloud.AgentMetrics
}

func (m PipelineModel) Update(msg tea.Msg) (PipelineModel, tea.Cmd) {
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

	case GotPipelineID:
		m.pipelineID = msg.PipelineID
		return m, m.loadData(m.ctx, true, false)

	case GotPipelineError:
		m.loading = false
		m.err = msg.Err
		if m.err == nil {
			return m, m.ReloadData
		}
		return m, nil

	case GotPipeline:
		m.pipeline = msg.Pipeline
		m.pipelineID = msg.Pipeline.ID
		return m, m.loadData(m.ctx, false, false)

	case ReloadPipelineDataRequested:
		var ctx context.Context
		ctx, m.cancelFunc = context.WithCancel(m.ctx)
		return m, tea.Tick(time.Second*5, func(time.Time) tea.Msg {
			return m.loadData(ctx, true, true)()
		})

	case GotPipelineData:
		m.loading = false
		m.err = nil
		if msg.WithPipeline {
			m.pipeline = msg.Pipeline
			m.pipelineID = msg.Pipeline.ID
		}
		m.tableRows = agentMetricsToTableRows(msg.PipelineMetrics)
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

func (m PipelineModel) View() string {
	if m.loading {
		return "Loading data... please wait"
	}

	if err := m.err; err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(fmt.Sprintf("Pipeline %q metrics", m.pipeline.Name)),
		m.viewMetrics(),
	)
}

func (m PipelineModel) viewMetrics() string {
	if len(m.tableRows) == 0 {
		return "No metrics"
	}

	return m.table.View()
}
