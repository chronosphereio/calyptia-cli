package main

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	cloud "github.com/calyptia/api/types"
)

func newCmdTop(config *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Display metrics",
	}

	cmd.AddCommand(
		newCmdTopProject(config),
		newCmdTopAgent(config),
		newCmdTopPipeline(config),
	)

	return cmd
}

func initialProjectModel(ctx context.Context, cloud Client, projectID string, metricsStart, metricsInterval time.Duration, last uint) Model {
	return Model{
		currentView: "project",
		project:     NewProjectModel(ctx, cloud, projectID, metricsStart, metricsInterval, last),
		agent:       NewAgentModel(ctx, cloud, projectID, "", metricsStart, metricsInterval),
	}
}

func initialAgentModel(ctx context.Context, cloud Client, projectID, agentKey string, metricsStart, metricsInterval time.Duration) Model {
	return Model{
		currentView: "agent",
		agent:       NewAgentModel(ctx, cloud, projectID, agentKey, metricsStart, metricsInterval),
	}
}

func initialPipelineModel(ctx context.Context, cloud Client, projectID, pipelineKey string, metricsStart, metricsInterval time.Duration) Model {
	return Model{
		currentView: "pipeline",
		pipeline:    NewPipelineModel(ctx, cloud, projectID, pipelineKey, metricsStart, metricsInterval),
	}
}

type Model struct {
	currentView string
	agent       AgentModel
	project     ProjectModel
	pipeline    PipelineModel
}

func (m Model) Init() tea.Cmd {
	switch m.currentView {
	case "project":
		return m.project.Init()
	case "agent":
		return m.agent.Init()
	case "pipeline":
		return m.pipeline.Init()
	}
	return nil
}

func NavigateBackToProject() tea.Msg {
	return WentBackToProject{}
}

type WentBackToProject struct{}

func NavigateToAgent(agent cloud.Agent, metrics cloud.AgentMetrics) tea.Cmd {
	return func() tea.Msg {
		return WentToAgent{
			Agent:   agent,
			Metrics: metrics,
		}
	}
}

type WentToAgent struct {
	Agent   cloud.Agent
	Metrics cloud.AgentMetrics
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case WentToAgent:
		m.agent.SetBackEnabled(true)
		m.agent.SetData(msg.Agent, msg.Metrics)
		m.currentView = "agent"
		return m, m.agent.ReloadData

	case WentBackToProject:
		m.currentView = "project"
		return m, m.project.ReloadData
	}

	switch m.currentView {
	case "project":
		var cmd tea.Cmd
		m.project, cmd = m.project.Update(msg)
		return m, cmd
	case "agent":
		var cmd tea.Cmd
		m.agent, cmd = m.agent.Update(msg)
		return m, cmd
	case "pipeline":
		var cmd tea.Cmd
		m.pipeline, cmd = m.pipeline.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.currentView {
	case "project":
		return m.project.View()
	case "agent":
		return m.agent.View()
	case "pipeline":
		return m.pipeline.View()
	}

	return "Nothing to see here"
}
