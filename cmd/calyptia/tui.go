package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/calyptia/cloud"
	"github.com/calyptia/cloud-cli/auth0"
	cloudclient "github.com/calyptia/cloud-cli/cloud"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hako/durafmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/term"
)

const rate = time.Second * 30

// keyMap defines a set of keybindings. To work for help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type keyMap struct {
	Enter  key.Binding
	Back   key.Binding
	Logout key.Binding
	Help   key.Binding
	Quit   key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.Logout}, // first column
		{k.Help, k.Quit},   // second column
	}
}

var keys = keyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
		key.WithDisabled(),
	),
	Back: key.NewBinding(
		key.WithKeys("alt+left", "ctrl+[", "backspace"),
		key.WithHelp("backspace", "go back"),
		key.WithDisabled(),
	),
	Logout: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "logout"),
		key.WithDisabled(),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

type model struct {
	ctx context.Context

	keys    keyMap
	help    help.Model
	spinner spinner.Model

	auth0 *auth0.Client
	cloud *cloudclient.Client

	requestingDeviceCode bool
	errDeviceCode        error
	deviceCode           auth0.DeviceCode

	fetchingAccessToken bool
	errAccessToken      error

	fetchingProjects bool
	errProjects      error
	showProjectList  bool
	projectList      list.Model

	fetchingMetrics  bool
	errMetrics       error
	showMetricsTable bool
	metrics          cloud.ProjectMetrics

	fetchingAgents bool
	errAgents      error
	showAgentList  bool
	agentList      list.Model
}

func (m model) Init() tea.Cmd {
	if m.requestingDeviceCode {
		return tea.Batch(
			m.requestDeviceCode,
			spinner.Tick,
		)
	}

	return tea.Batch(
		m.fetchProjects,
		spinner.Tick,
	)
}

type requestDeviceCodeFailed struct {
	err error
}

type requestDeviceCodeOK struct {
	deviceCode auth0.DeviceCode
}

func (m model) requestDeviceCode() tea.Msg {
	dc, err := m.auth0.DeviceCode(m.ctx)
	if err != nil {
		return requestDeviceCodeFailed{err}
	}

	return requestDeviceCodeOK{dc}
}

type fetchAccessTokenFailed struct {
	err error
}

type fetchAccessTokenOK struct {
	accessToken *oauth2.Token
}

type refetchAccessToken struct{}

func (m model) fetchAccessToken() tea.Msg {
	at, err := m.auth0.AccessToken(m.ctx, m.deviceCode.DeviceCode)
	if auth0.IsAuthorizationPendingError(err) {
		return refetchAccessToken{}
	}

	if err != nil {
		return fetchAccessTokenFailed{err}
	}

	return fetchAccessTokenOK{at}
}

type fetchProjectsFailed struct {
	err error
}

type fetchProjectsOK struct {
	projects []cloud.Project
}

func (m model) fetchProjects() tea.Msg {
	pp, err := m.cloud.Projects(m.ctx, 0)
	if err != nil {
		return fetchProjectsFailed{err}
	}

	return fetchProjectsOK{pp}
}

type fetchMetricsFailed struct {
	err error
}

type fetchMetricsOK struct {
	metrics   cloud.ProjectMetrics
	projectID string
	isRefetch bool
}

func (m model) fetchMetrics(projectID string, isRefetch bool) tea.Cmd {
	return func() tea.Msg {
		mm, err := m.cloud.Metrics(m.ctx, projectID, rate*-2, rate)
		if err != nil {
			if isRefetch {
				return nil
			}

			return fetchMetricsFailed{err}
		}

		return fetchMetricsOK{
			metrics:   mm,
			projectID: projectID,
			isRefetch: isRefetch,
		}
	}
}

type fetchAgentsFailed struct {
	err error
}

type fetchAgentsOK struct {
	agents []cloud.Agent
}

func (m model) fetchAgents(projectID string) tea.Cmd {
	return func() tea.Msg {
		aa, err := m.cloud.Agents(m.ctx, projectID, 0)
		if err != nil {
			return fetchAgentsFailed{err}
		}

		return fetchAgentsOK{aa}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.showProjectList {
			top, right, bottom, left := docStyle.GetPadding()
			m.projectList.SetSize(msg.Width-left-right, msg.Height-top-bottom)
		} else if m.showAgentList {
			top, right, bottom, left := docStyle.GetPadding()
			m.agentList.SetSize(msg.Width-left-right, msg.Height-top-bottom)
		} else {
			m.help.Width = msg.Width
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Enter):
			if m.showProjectList {
				v, ok := m.projectList.SelectedItem().(projectListItem)
				if ok {
					m.keys.Back.SetEnabled(true)
					m.keys.Enter.SetEnabled(false)
					m.showProjectList = false
					m.fetchingMetrics = true
					m.fetchingAgents = true
					m.metrics = cloud.ProjectMetrics{
						Measurements: map[string]cloud.ProjectMeasurement{},
						TopPlugins:   cloud.PluginTotal{},
					}
					cmds = append(cmds,
						m.fetchMetrics(v.Project.ID, false),
						m.fetchAgents(v.Project.ID),
						spinner.Tick,
					)
				}
			}
		case key.Matches(msg, m.keys.Back):
			if m.showMetricsTable {
				m.keys.Back.SetEnabled(false)
				m.keys.Enter.SetEnabled(true)

				m.showProjectList = true

				m.showMetricsTable = false
				m.fetchingMetrics = false
				m.errMetrics = nil

				m.showAgentList = false
				m.fetchingAgents = false
				m.errAgents = nil
			}
		case key.Matches(msg, m.keys.Logout):
			_ = deleteAccessToken()
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	case requestDeviceCodeFailed:
		m.requestingDeviceCode = false
		m.errDeviceCode = msg.err
	case requestDeviceCodeOK:
		m.requestingDeviceCode = false
		m.errDeviceCode = nil

		m.deviceCode = msg.deviceCode

		m.fetchingAccessToken = true
		m.errAccessToken = nil

		_ = browser.OpenURL(msg.deviceCode.VerificationURIComplete)

		cmds = append(cmds,
			tea.Tick(time.Duration(msg.deviceCode.Interval)*time.Second, func(time.Time) tea.Msg {
				return m.fetchAccessToken()
			}),
			spinner.Tick,
		)
	case refetchAccessToken:
		cmds = append(cmds,
			tea.Tick(time.Duration(m.deviceCode.Interval)*time.Second, func(time.Time) tea.Msg {
				return m.fetchAccessToken()
			}),
			spinner.Tick,
		)
	case fetchAccessTokenFailed:
		m.fetchingAccessToken = false
		m.errAccessToken = msg.err
	case fetchAccessTokenOK:
		m.fetchingAccessToken = false
		m.errAccessToken = nil

		saveAccessToken(msg.accessToken)

		m.cloud.HTTPClient = m.auth0.Client(m.ctx, msg.accessToken)

		m.fetchingProjects = true

		cmds = append(cmds,
			m.fetchProjects,
			spinner.Tick,
		)
	case fetchProjectsFailed:
		m.fetchingProjects = false
		m.errProjects = msg.err
	case fetchProjectsOK:
		m.fetchingProjects = false
		m.errProjects = nil

		m.keys.Enter.SetEnabled(true)
		m.keys.Back.SetEnabled(true)

		items := make([]list.Item, len(msg.projects))
		for i, p := range msg.projects {
			items[i] = projectListItem{p}
		}

		projectList := list.NewModel(items, list.NewDefaultDelegate(), 0, 0)
		projectList.Title = "Projects"
		projectList.AdditionalFullHelpKeys = func() []key.Binding {
			return []key.Binding{m.keys.Enter, m.keys.Logout}
		}

		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			top, right, bottom, left := docStyle.GetPadding()
			projectList.SetSize(w-left-right, h-top-bottom)
		}

		m.showProjectList = true
		m.projectList = projectList
	case fetchMetricsFailed:
		m.fetchingMetrics = false
		m.errMetrics = msg.err
	case fetchMetricsOK:
		m.fetchingMetrics = false
		m.errMetrics = nil
		if !msg.isRefetch {
			m.showMetricsTable = true
		}
		if len(msg.metrics.Measurements) != 0 {
			m.metrics = msg.metrics
		} else if m.metrics.Measurements == nil {
			m.metrics = cloud.ProjectMetrics{
				Measurements: map[string]cloud.ProjectMeasurement{},
				TopPlugins:   cloud.PluginTotal{},
			}
		}

		if m.showMetricsTable {
			cmds = append(cmds,
				tea.Tick(time.Second, func(t time.Time) tea.Msg {
					return m.fetchMetrics(msg.projectID, true)()
				}),
			)
		}
	case fetchAgentsFailed:
		m.fetchingAgents = false
		m.errAgents = msg.err
	case fetchAgentsOK:
		m.fetchingAgents = false
		m.errAgents = nil

		m.keys.Back.SetEnabled(true)

		items := make([]list.Item, len(msg.agents))
		for i, a := range msg.agents {
			items[i] = agentListItem{a}
		}

		agentList := list.NewModel(items, list.NewDefaultDelegate(), 0, 0)
		agentList.Title = "Agents"
		agentList.AdditionalFullHelpKeys = func() []key.Binding {
			return []key.Binding{m.keys.Logout}
		}

		w, h, err := term.GetSize(int(os.Stdout.Fd()))
		if err == nil {
			top, right, bottom, left := docStyle.GetPadding()
			agentList.SetSize(w-left-right, h-top-bottom)
		}

		m.agentList = agentList
		m.showAgentList = true
	}

	if m.showProjectList {
		var cmd tea.Cmd
		m.projectList, cmd = m.projectList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.showAgentList {
		var cmd tea.Cmd
		m.agentList, cmd = m.agentList.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.requestingDeviceCode || m.fetchingAccessToken || m.fetchingProjects || m.fetchingMetrics || m.fetchingAgents {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	if cmds == nil {
		return m, nil
	}

	if len(cmds) == 0 {
		return m, cmds[0]
	}

	return m, tea.Batch(cmds...)
}

var (
	docStyle   = lipgloss.NewStyle().Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 1)
)

func (m model) View() string {
	screenWidth, screenHeight, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		screenWidth = 80
		screenHeight = 60
	}

	_ = screenWidth

	var doc strings.Builder

	if m.requestingDeviceCode {
		doc.WriteString(m.spinner.View())
		doc.WriteString(" Requesting a new device code... please wait.")
	}

	if err := m.errDeviceCode; err != nil {
		doc.WriteString("Could not request device code: " + err.Error())
	}

	if m.fetchingAccessToken {
		doc.WriteString(fmt.Sprintf("Please go to %s to authorize this application.\n\n", m.deviceCode.VerificationURIComplete))
		doc.WriteString(m.spinner.View())
		doc.WriteString(" Waiting authorization...")
	}

	if err := m.errAccessToken; err != nil {
		doc.WriteString("Could not fetch access token: " + err.Error())
	}

	if m.fetchingProjects {
		doc.WriteString(m.spinner.View())
		doc.WriteString(" Fetching your project list... please wait.")
	}

	if err := m.errProjects; err != nil {
		doc.WriteString("Could not fetch your project list: " + err.Error())
	}

	if m.showProjectList {
		doc.WriteString(m.projectList.View())
	}

	if m.fetchingMetrics {
		doc.WriteString(m.spinner.View())
		doc.WriteString(" Fetching metrics... please wait.")
	}

	if err := m.errMetrics; err != nil {
		doc.WriteString("Could not fetch metrics: " + err.Error())
	}

	if m.showMetricsTable {
		if len(m.metrics.Measurements) == 0 {
			doc.WriteString(titleStyle.Copy().MarginLeft(2).Render("Overview") + "\n\n")
			doc.WriteString("No measurements")
		} else {
			tw := table.NewWriter()
			tw.Style().Box = table.StyleBoxRounded
			tw.AppendHeader(table.Row{"Plugin", "Metric", "Value"})
			first := true

			for _, measurementName := range measurementNames(m.metrics.Measurements) {
				measurement := m.metrics.Measurements[measurementName]

				if !first {
					tw.AppendSeparator()
				}
				if first {
					first = false
				}

				for _, pluginName := range pluginNames(measurement.Plugins) {
					plugin := measurement.Plugins[pluginName]

					// skip internal plugins.
					if strings.HasPrefix(pluginName, "calyptia.") || strings.HasPrefix(pluginName, "fluentbit_metrics.") {
						continue
					}

					for _, metricName := range metricNames(plugin.Metrics) {
						points := plugin.Metrics[metricName]

						d := len(points)
						if d < 2 {
							tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), metricName, "Not enough data"})
							continue
						}

						var val *float64
						for i := d - 1; i > 0; i-- {
							curr := points[i].Value
							prev := points[i-1].Value

							if curr == nil || prev == nil {
								continue
							}

							secs := rate.Seconds()
							v := (*curr / secs) - (*prev / secs)
							val = &v
							break
						}

						if val == nil {
							tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), metricName, "No data"})
							continue
						}

						var s string
						if metricName == "bytes_total" || metricName == "proc_bytes_total" {
							s = bytefmt.ByteSize(uint64(math.Round(*val)))
						} else {
							s = fmtFloat64(*val)
						}
						tw.AppendRow(table.Row{fmt.Sprintf("%s (%s)", pluginName, measurementName), metricName, s})
					}
				}
			}

			doc.WriteString(titleStyle.Copy().MarginLeft(2).Render("Overview") + "\n")
			doc.WriteString(tw.Render())
		}
	}

	if m.fetchingAgents {
		doc.WriteString("\n\n")
		doc.WriteString(m.spinner.View())
		doc.WriteString(" Fetching your agent list... please wait.")
	}

	if err := m.errAgents; err != nil {
		doc.WriteString("\n\n")
		doc.WriteString("Could not fetch your agent list: " + err.Error())
	}

	if m.showAgentList {
		doc.WriteString("\n\n")

		docHeight := lipgloss.Height(doc.String())
		top, _, bottom, _ := docStyle.GetPadding()

		m.agentList.SetSize(screenWidth, screenHeight-docHeight-top-bottom)
		doc.WriteString(m.agentList.View())
	}

	// list view already comes with help.
	if !m.showProjectList && !m.showAgentList {
		helpView := m.help.View(m.keys)
		helpViewHeight := lipgloss.Height(helpView)

		docHeight := lipgloss.Height(doc.String())
		top, _, bottom, _ := docStyle.GetPadding()

		spaceHeight := screenHeight - docHeight - helpViewHeight - top - bottom
		// prevent nagative number for strings.Repeat later
		if spaceHeight < 1 {
			spaceHeight = 1
		}
		doc.WriteString(
			strings.Repeat("\n", spaceHeight) + helpView,
		)
	}

	return docStyle.Render(doc.String())
}

type projectListItem struct {
	cloud.Project
}

func (item projectListItem) FilterValue() string { return item.Project.Name }
func (item projectListItem) Title() string       { return item.Project.Name }
func (item projectListItem) Description() string { return item.Project.ID }

type agentListItem struct {
	cloud.Agent
}

func (item agentListItem) FilterValue() string { return item.Agent.Name }
func (item agentListItem) Title() string       { return item.Agent.Name }
func (item agentListItem) Description() string {
	out := fmt.Sprintf("%s %s", item.Agent.Type, item.Agent.Version)
	if item.Agent.LastMetricsAddedAt.IsZero() {
		out += " (inactive)"
	} else if item.Agent.LastMetricsAddedAt.Before(time.Now().Add(rate * -2)) {
		out += fmt.Sprintf(" (inactive for %s)", durafmt.ParseShort(time.Since(item.Agent.LastMetricsAddedAt)).LimitFirstN(1).String())
	} else {
		out += " (active)"
	}

	return out
}

func measurementNames(m map[string]cloud.ProjectMeasurement) []string {
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
