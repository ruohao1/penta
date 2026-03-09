package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ruohao1/penta/internal/events"
	"github.com/ruohao1/penta/internal/tui/messages"
	"github.com/ruohao1/penta/internal/tui/styles"
)

type logPaneView struct {
	viewport viewport.Model
	lines    []string

	width, height int
	active        bool
	focused       bool
}

func (v *logPaneView) Init() tea.Cmd {
	// TODO: implement log pane
	return nil
}

func (v *logPaneView) Update(msg tea.Msg) (Pane, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case messages.PaneSizeMsg:
		v.SetSize(msg.Width, msg.Height)
		return v, nil
	case events.Event:
		v.appendLine(formatEventLine(msg))
		v.viewport.GotoBottom()
		return v, nil
	}

	_, cmd = v.viewport.Update(msg)

	return v, cmd
}

func (v logPaneView) View() tea.View {
	var view tea.View
	var style lipgloss.Style

	switch {
	case v.focused:
		style = styles.FocusedPaneStyle
	case v.active:
		style = styles.ActivePaneStyle
	default:
		style = styles.DefaultPaneStyle
	}

	content := styles.WithTitle(style, "Logs", v.viewport.View(), v.width, v.height)

	view.Content = content

	return view
}

func LogPane() *logPaneView {
	vp := viewport.New()
	vp.SetContentLines([]string{"[INFO] waiting for events..."})
	return &logPaneView{viewport: vp, lines: []string{"[INFO] waiting for events..."}, active: false, focused: false}
}

func (v logPaneView) Focused() bool {
	return v.focused
}

func (v *logPaneView) SetFocused(focused bool) {
	v.focused = focused
}

func (v logPaneView) Active() bool {
	return v.active
}

func (v *logPaneView) SetActive(active bool) {
	v.active = active
}

func (v logPaneView) Size() (int, int) {
	return v.width, v.height
}

func (v *logPaneView) SetSize(w, h int) {
	v.width = w
	v.height = h
	v.viewport.SetWidth(max(1, w-2))
	v.viewport.SetHeight(max(1, h-2))
}

func (v logPaneView) Help() string {
	return "Log pane help: (not implemented yet)"
}

func (v *logPaneView) appendLine(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	v.lines = append(v.lines, line)
	const maxLogLines = 500
	if len(v.lines) > maxLogLines {
		v.lines = v.lines[len(v.lines)-maxLogLines:]
	}
	v.viewport.SetContentLines(v.lines)
}

func formatEventLine(ev events.Event) string {
	ts := ev.At
	if ts.IsZero() {
		ts = time.Now()
	}
	if ev.Kind == events.Finding {
		parts := []string{ts.Format("15:04:05"), "FIND"}
		if status, ok := ev.Data["status_code"]; ok {
			parts = append(parts, fmt.Sprintf("status=%v", status))
		}
		if url, ok := ev.Data["url"].(string); ok && url != "" {
			parts = append(parts, "url="+url)
		}
		if depth, ok := ev.Data["depth"]; ok {
			parts = append(parts, fmt.Sprintf("depth=%v", depth))
		}
		if size, ok := ev.Data["content_length"]; ok {
			parts = append(parts, fmt.Sprintf("size=%v", size))
		}
		if latency, ok := ev.Data["duration_ms"]; ok {
			parts = append(parts, fmt.Sprintf("ms=%v", latency))
		}
		if ev.Err != "" {
			parts = append(parts, "err="+ev.Err)
		}
		return strings.Join(parts, " ")
	}

	parts := []string{ts.Format("15:04:05"), string(ev.Kind)}
	if ev.Stage != "" {
		parts = append(parts, "stage="+ev.Stage)
	}
	if ev.Target != "" {
		parts = append(parts, "target="+ev.Target)
	}
	if ev.Err != "" {
		parts = append(parts, "err="+ev.Err)
	}
	return strings.Join(parts, " ")
}
