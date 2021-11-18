package main

import (
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
)

func NewModel(opts ...Opt) *Model {
	m := &Model{}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

type Opt func(*Model)

func WithProject(mp *ProjectModel) Opt {
	return func(m *Model) {
		m.projectModel = mp
	}
}

type Model struct {
	currentView View
	viewHistory []View

	projectModel *ProjectModel
}

type WentToProjectViewMsg struct{}

func GoToProjectView() tea.Msg {
	return WentToProjectViewMsg{}
}

func (m *Model) Init() tea.Cmd {
	if m.projectModel != nil {
		return tea.Batch(
			m.projectModel.Init(),
			GoToProjectView,
		)
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case WentToProjectViewMsg:
		m.currentView = ViewProject
		m.viewHistory = append(m.viewHistory, m.currentView)
		return m, nil
	}

	switch m.currentView {
	case ViewProject:
		var cmd tea.Cmd
		m.projectModel, cmd = m.projectModel.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) View() string {
	switch m.currentView {
	case ViewProject:
		return m.projectModel.View()
	}
	return ""
}
