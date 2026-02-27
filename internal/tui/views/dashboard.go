package views

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Ruohao1/penta/internal/tui/components"
)

type dashboardPane int

const (
	streamPane dashboardPane = iota
	logPane
)

type dashboard struct {
	focusedPane dashboardPane
	focusing    bool

	panes map[dashboardPane]components.Pane

	width, height int
}

func DashboardModel() tea.Model {
	columns := []components.Column{
		{Title: "Rank", WidthPercent: 20},
		{Title: "City", WidthPercent: 20},
		{Title: "Country", WidthPercent: 30},
		{Title: "Population", WidthPercent: 30},
	}

	panes := map[dashboardPane]components.Pane{
		streamPane: components.TablePane(columns),
		logPane:    components.TablePane(columns),
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

		paneOuterH := max(1, m.height-2)
		paneInnerH := max(1, paneOuterH-paneStyle.GetVerticalFrameSize())

		leftOuterW := m.width * 30 / 100
		rightOuterW := m.width - leftOuterW

		leftW := max(1, leftOuterW-paneStyle.GetHorizontalFrameSize())
		rightW := max(1, rightOuterW-paneStyle.GetHorizontalFrameSize())

		m.panes[logPane].SetSize(leftW, paneInnerH)
		m.panes[streamPane].SetSize(rightW, paneInnerH)
		return m, nil
	}

	if m.focusing {
		if key, ok := msg.(tea.KeyPressMsg); ok {
			if key.String() == "esc" || key.String() == "escape" {
				m.focusing = false
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

	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab":
			n := len(m.panes)
			if n > 0 {
				m.focusedPane = dashboardPane((int(m.focusedPane) + 1) % n)
			}
		case "shift+tab":
			n := len(m.panes)
			if n > 0 {
				m.focusedPane = dashboardPane((int(m.focusedPane) - 1 + n) % n)
			}
		case "enter":
			m.focusing = true
		case "p":
			leftw, lefth := m.panes[logPane].Size()
			rightw, righth := m.panes[streamPane].Size()

			return m, tea.Printf("log pane size: %dx%d, stream pane size: %dx%d", leftw, lefth, rightw, righth)
		}
	}

	return m, nil
}

var (
	topBarStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62"))

	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			Padding(0, 0)

	activePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("69")).
			Padding(0, 0)

	selectedPaneStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("205")).
				Bold(true).
				Padding(0, 0)
)

func (m dashboard) View() tea.View {
	w, h := m.width, m.height
	if w <= 0 || h <= 0 {
		return tea.NewView("loading...")
	}

	topH := 1
	helpH := 1
	bodyH := max(1, h-topH-helpH)

	leftW := w * 30 / 100
	rightW := w - leftW

	topBar := topBarStyle.Width(w).Height(topH).Render("Penta Dashboard")

	streamView := m.panes[streamPane].View().Content
	logView := m.panes[logPane].View().Content

	streamPaneView := paneStyle.Width(rightW).Height(bodyH).Render(streamView)
	logPaneView := paneStyle.Width(leftW).Height(bodyH).Render(logView)

	switch m.focusedPane {
	case streamPane:
		streamPaneView = activePaneStyle.Width(rightW).Height(bodyH).Render(streamView)
	case logPane:
		logPaneView = activePaneStyle.Width(leftW).Height(bodyH).Render(logView)
	}
	help := "tab: next pane | shift-tab: prev pane | enter: focus pane | esc: unfocus pane"
	if m.focusing {
		switch m.focusedPane {
		case streamPane:
			streamPaneView = selectedPaneStyle.Width(rightW).Height(bodyH).Render(streamView)
		case logPane:
			logPaneView = selectedPaneStyle.Width(leftW).Height(bodyH).Render(logView)
		}
		help = m.panes[m.focusedPane].Help()
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, logPaneView, streamPaneView)
	return tea.NewView(
		lipgloss.JoinVertical(lipgloss.Left, topBar, body, help))
}

// func (m dashboard) leftOuterWidth() int {
// 	return m.width * 30 / 100
// }
//
// func (m dashboard) rightOuterWidth() int {
// 	return m.width - m.leftOuterWidth()
// }
