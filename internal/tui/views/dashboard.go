package views

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Ruohao1/penta/internal/core/events"
	"github.com/Ruohao1/penta/internal/tui/components"
	"github.com/Ruohao1/penta/internal/tui/styles"
)

type dashboardPane int

const (
	streamPane dashboardPane = iota
	logPane
	statusPane
)

type dashboard struct {
	focusedPane dashboardPane
	focusing    bool

	panes map[dashboardPane]components.Pane

	width, height int
}

func DashboardModel() tea.Model {
	columns := []components.Column{
		{Title: "Status", WidthPercent: 20},
		{Title: "City", WidthPercent: 20},
		{Title: "Country", WidthPercent: 30},
		{Title: "Population", WidthPercent: 30},
	}

	panes := map[dashboardPane]components.Pane{
		logPane:    components.LogPane(),
		streamPane: components.TablePane(columns),
		statusPane: components.TablePane(columns),
	}
	return dashboard{
		focusedPane: streamPane,
		focusing:    false,
		panes:       panes,
	}
}

func (m dashboard) Init() tea.Cmd {
	return nil
}

func (m dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

		bodyH := max(1, m.height-2)

		streamH := bodyH * 80 / 100
		logH := bodyH - streamH

		leftW := m.width * 30 / 100
		rightW := m.width - leftW

		m.panes[statusPane].SetSize(leftW, bodyH)
		m.panes[streamPane].SetSize(rightW, streamH)
		m.panes[logPane].SetSize(rightW, logH)
		return m, nil
	}

	if _, ok := msg.(events.Event); ok {
		cmds := make([]tea.Cmd, 0, len(m.panes))
		for key, child := range m.panes {
			updated, cmd := child.Update(msg)
			m.panes[key] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	if m.focusing {
		return m.handleFocusing(msg)
	}

	return m.handleDefault(msg)
}

func (m dashboard) handleFocusing(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "esc" || key.String() == "escape" {
			m.focusing = false
			m.panes[m.focusedPane].SetActive(false)
			return m, nil
		}
	}

	child, ok := m.panes[m.focusedPane]
	if !ok {
		return m, nil
	}
	updated, cmd := child.Update(msg)
	m.panes[m.focusedPane] = updated
	return m, cmd
}

func (m dashboard) handleDefault(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab":
			n := len(m.panes)
			m.panes[m.focusedPane].SetFocused(false)
			if n > 0 {
				m.focusedPane = dashboardPane((int(m.focusedPane) + 1) % n)
			}
			m.panes[m.focusedPane].SetFocused(true)
		case "shift+tab":
			n := len(m.panes)
			m.panes[m.focusedPane].SetFocused(false)
			if n > 0 {
				m.focusedPane = dashboardPane((int(m.focusedPane) - 1 + n) % n)
			}
			m.panes[m.focusedPane].SetFocused(true)
		case "enter":
			m.focusing = true
			m.panes[m.focusedPane].SetActive(true)

		case "p":
			leftw, lefth := m.panes[statusPane].Size()
			rightw, righth := m.panes[streamPane].Size()

			return m, tea.Printf("status pane size: %dx%d, stream pane size: %dx%d", leftw, lefth, rightw, righth)
		}
	}

	return m, nil
}

var (
	topBarStyle = lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1).
		Foreground(styles.ColorTextPrimary).
		Background(styles.ColorSurfaceAccent)
)

func (m dashboard) View() tea.View {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return tea.NewView("loading...")
	}

	topBar := topBarStyle.Width(w).Height(1).Render("Penta Dashboard")

	streamPaneView := m.panes[streamPane].View().Content
	statusPaneView := m.panes[statusPane].View().Content
	logPaneView := m.panes[logPane].View().Content

	help := "Tab: Switch Pane | Enter: Focus Pane | Esc: Unfocus Pane"
	if m.focusing {
		help = m.panes[m.focusedPane].Help()
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top, statusPaneView, lipgloss.JoinVertical(lipgloss.Left, streamPaneView, logPaneView))
	return tea.NewView(
		lipgloss.JoinVertical(lipgloss.Left, topBar, body, help))
}
