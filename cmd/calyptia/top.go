package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
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

type View string

const (
	ViewProject View = "project"
	ViewAgent   View = "agent"
)

type Model struct {
	currentView View
	viewHistory []View

	StartingProjectKey string
	StartingAgentKey   string

	ProjectModel *ProjectModel
	AgentModel   *AgentModel
}

type WentToProjectViewMsg struct {
	ProjectKey string
}

func GoToProjectView(projectKey string) tea.Cmd {
	return func() tea.Msg {
		return WentToProjectViewMsg{ProjectKey: projectKey}
	}
}

type WentToAgentViewMsg struct {
	AgentKey string
}

func GoToAgentView(agentKey string) tea.Cmd {
	return func() tea.Msg {
		return WentToAgentViewMsg{AgentKey: agentKey}
	}
}

func (m *Model) Init() tea.Cmd {
	switch {
	case m.StartingAgentKey != "":
		return GoToAgentView(m.StartingAgentKey)
	case m.StartingProjectKey != "":
		return GoToProjectView(m.StartingProjectKey)
	default:
		return nil
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "backspace":
			if d := len(m.viewHistory); d > 1 {
				m.currentView = m.viewHistory[d-2]
				m.viewHistory = append(m.viewHistory, m.currentView)
			}
			return m, nil
		}

	case WentToProjectViewMsg:
		m.currentView = ViewProject
		m.viewHistory = append(m.viewHistory, m.currentView)
		m.ProjectModel.ProjectKey = msg.ProjectKey
		m.ProjectModel.SetProjectID(msg.ProjectKey)
		return m, m.ProjectModel.Init()

	case WentToAgentViewMsg:
		m.currentView = ViewAgent
		m.viewHistory = append(m.viewHistory, m.currentView)
		m.AgentModel.AgentKey = msg.AgentKey
		m.AgentModel.SetAgentID(msg.AgentKey)
		return m, m.AgentModel.Init()
	}

	switch m.currentView {
	case ViewProject:
		var cmd tea.Cmd
		m.ProjectModel, cmd = m.ProjectModel.Update(msg)
		return m, cmd

	case ViewAgent:
		var cmd tea.Cmd
		m.AgentModel, cmd = m.AgentModel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	var doc strings.Builder
	switch m.currentView {
	case ViewProject:
		doc.WriteString(m.ProjectModel.View())
	case ViewAgent:
		doc.WriteString(m.AgentModel.View())
	}
	return docStyle.Render(doc.String())
}
