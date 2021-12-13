package table

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/juju/ansiterm/tabwriter"
	"golang.org/x/term"
)

type Row interface {
	// Render the row into the given tabwriter.
	Render(w io.Writer, model Model, index int)
}

func NewModel(cols []string) Model {
	width, _, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = 32
	}
	vp := viewport.Model{
		Width:  width,
		Height: 1,
	}
	tw := &tabwriter.Writer{}
	return Model{
		cols:      cols,
		header:    strings.Join(cols, " "),
		viewPort:  vp,
		tabWriter: tw,
	}
}

type Model struct {
	cols       []string
	rows       []Row
	header     string
	viewPort   viewport.Model
	tabWriter  *tabwriter.Writer
	navEnabled bool
	cursor     int
}

func (m *Model) SetSize(width, height int) {
	m.viewPort.Width = width
	m.viewPort.Height = height - 1
}

func (m *Model) SetNavEnabled(enabled bool) {
	m.navEnabled = enabled
}

func (m Model) Cursor() int {
	return m.cursor
}

func (m Model) SelectedRow() Row {
	return m.rows[m.cursor]
}

func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	m.updateView()
}

func (m *Model) updateView() {
	var b strings.Builder
	m.tabWriter.Init(&b, 0, 4, 1, ' ', 0)
	fmt.Fprintln(m.tabWriter, strings.Join(m.cols, "\t"))
	for i, row := range m.rows {
		row.Render(m.tabWriter, *m, i)
	}
	m.tabWriter.Flush()

	// split table at first line-break to take header and rows apart.
	parts := strings.SplitN(b.String(), "\n", 2)
	if len(parts) != 0 {
		m.header = parts[0]
		if len(parts) == 2 {
			m.viewPort.SetContent(parts[1])
		}
	}
}

func (m Model) CursorIsAtTop() bool {
	return m.cursor == 0
}

func (m Model) CursorIsAtBottom() bool {
	return m.cursor == len(m.rows)-1
}

func (m Model) CursorIsPastBottom() bool {
	return m.cursor >= len(m.rows)-1
}

func (m *Model) GoUp() {
	if m.CursorIsAtTop() {
		return
	}

	m.cursor--
	m.updateView()

	if m.cursor < m.viewPort.YOffset {
		m.viewPort.LineUp(1)
	}
}

func (m *Model) GoDown() {
	if m.CursorIsAtBottom() {
		return
	}

	m.cursor++
	m.updateView()

	if m.cursor > m.viewPort.YOffset+m.viewPort.Height-1 {
		m.viewPort.LineDown(1)
	}
}

func (m *Model) GoPageUp() {
	if m.CursorIsAtTop() {
		return
	}

	m.cursor -= m.viewPort.Height
	if m.cursor < 0 {
		m.cursor = 0
	}

	m.updateView()

	m.viewPort.ViewUp()
}

func (m *Model) GoPageDown() {
	if m.CursorIsAtBottom() {
		return
	}

	m.cursor += m.viewPort.Height
	if m.CursorIsPastBottom() {
		m.cursor = len(m.rows) - 1
	}

	m.updateView()

	m.viewPort.ViewDown()
}

func (m *Model) GoTop() {
	if m.CursorIsAtTop() {
		return
	}

	m.cursor = 0
	m.updateView()
	m.viewPort.GotoTop()
}

func (m *Model) GoBottom() {
	if m.CursorIsAtBottom() {
		return
	}

	m.cursor = len(m.rows) - 1
	m.updateView()
	m.viewPort.GotoBottom()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	// Viewport scrolling is managed by its own only if navigation is not enabled.
	// Otherwise we take care of it.
	if !m.navEnabled {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			// Extra cases for missing "home" and "end" keys from viewport bubble.
			switch msg.String() {
			case "home":
				lines := m.viewPort.GotoTop()
				if m.viewPort.HighPerformanceRendering {
					return m, viewport.ViewUp(m.viewPort, lines)
				}

				return m, nil
			case "end":
				lines := m.viewPort.GotoBottom()
				if m.viewPort.HighPerformanceRendering {
					return m, viewport.ViewDown(m.viewPort, lines)
				}

				return m, nil
			}
		}

		var cmd tea.Cmd
		m.viewPort, cmd = m.viewPort.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.GoUp()
		case "down":
			m.GoDown()
		case "pgup":
			m.GoPageUp()
		case "pgdown":
			m.GoPageDown()
		case "home":
			m.GoTop()
		case "end":
			m.GoBottom()
		}
	}

	return m, nil
}

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.header,
		m.viewPort.View(),
	)
}
