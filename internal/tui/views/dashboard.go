package views

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/tui/components"
	"github.com/ruohao1/penta/internal/tui/styles"
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

	totalFindings int
	errorFindings int
	status2xx     int
	status3xx     int
	status4xx     int
	status5xx     int
	streamRows    [][]string

	width, height int
}

const maxStreamRows = 500

func DashboardModel() tea.Model {
	streamColumns := []components.Column{
		{Title: "Status", WidthPercent: 10},
		{Title: "URL", WidthPercent: 44},
		{Title: "Depth", WidthPercent: 8},
		{Title: "Size", WidthPercent: 10},
		{Title: "ms", WidthPercent: 8},
		{Title: "Error", WidthPercent: 20},
	}
	statusColumns := []components.Column{
		{Title: "Metric", WidthPercent: 62},
		{Title: "Value", WidthPercent: 38},
	}

	panes := map[dashboardPane]components.Pane{
		logPane:    components.LogPane(),
		streamPane: components.TablePane(streamColumns),
		statusPane: components.TablePane(statusColumns),
	}
	if statusTable, ok := panes[statusPane].(components.TableRows); ok {
		statusTable.SetStringRows(initialStatusRows())
	}
	if streamTable, ok := panes[streamPane].(components.TableRows); ok {
		streamTable.SetStringRows([][]string{})
	}
	return dashboard{
		focusedPane: streamPane,
		focusing:    false,
		panes:       panes,
		streamRows:  [][]string{},
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
		m.handleEvent(msg.(events.Event))
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

func (m *dashboard) handleEvent(ev events.Event) {
	if ev.Kind != events.Finding {
		return
	}
	m.totalFindings++
	statusCode := intData(ev.Data, "status_code")
	if statusCode >= 200 && statusCode < 300 {
		m.status2xx++
	}
	if statusCode >= 300 && statusCode < 400 {
		m.status3xx++
	}
	if statusCode >= 400 && statusCode < 500 {
		m.status4xx++
	}
	if statusCode >= 500 {
		m.status5xx++
	}
	errText := strings.TrimSpace(ev.Err)
	if errText == "" {
		errText = stringData(ev.Data, "error")
	}
	if errText != "" {
		m.errorFindings++
	}

	if streamTable, ok := m.panes[streamPane].(components.TableRows); ok {
		m.streamRows = append(m.streamRows, []string{
			statusCell(statusCode, errText),
			stringDataOr(ev.Data, "url", ev.Target),
			strconv.Itoa(intData(ev.Data, "depth")),
			fmt.Sprint(anyData(ev.Data, "content_length")),
			fmt.Sprint(anyData(ev.Data, "duration_ms")),
			errText,
		})
		if len(m.streamRows) > maxStreamRows {
			m.streamRows = m.streamRows[len(m.streamRows)-maxStreamRows:]
		}
		streamTable.SetStringRows(m.streamRows)
	}

	if statusTable, ok := m.panes[statusPane].(components.TableRows); ok {
		statusTable.SetStringRows([][]string{
			{"Findings", strconv.Itoa(m.totalFindings)},
			{"Errors", strconv.Itoa(m.errorFindings)},
			{"2xx", strconv.Itoa(m.status2xx)},
			{"3xx", strconv.Itoa(m.status3xx)},
			{"4xx", strconv.Itoa(m.status4xx)},
			{"5xx", strconv.Itoa(m.status5xx)},
		})
	}
}

func initialStatusRows() [][]string {
	return [][]string{
		{"Findings", "0"},
		{"Errors", "0"},
		{"2xx", "0"},
		{"3xx", "0"},
		{"4xx", "0"},
		{"5xx", "0"},
	}
}

func anyData(data map[string]any, key string) any {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok {
		return ""
	}
	return v
}

func stringData(data map[string]any, key string) string {
	v := anyData(data, key)
	s, _ := v.(string)
	return s
}

func stringDataOr(data map[string]any, key, fallback string) string {
	v := strings.TrimSpace(stringData(data, key))
	if v == "" {
		return fallback
	}
	return v
}

func intData(data map[string]any, key string) int {
	v := anyData(data, key)
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case uint64:
		return int(x)
	case float64:
		return int(x)
	case string:
		n, err := strconv.Atoi(x)
		if err == nil {
			return n
		}
	}
	return 0
}

func statusCell(code int, errText string) string {
	if strings.TrimSpace(errText) != "" {
		return "ERR"
	}
	if code <= 0 {
		return "-"
	}
	return strconv.Itoa(code)
}
