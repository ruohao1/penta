package components

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Ruohao1/penta/internal/tui/messages"
	"github.com/Ruohao1/penta/internal/tui/styles"
)

type logPaneView struct {
	viewport viewport.Model

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
	// Mock log content
	vp.SetContentLines([]string{
		"[INFO] Application started",
		"[DEBUG] Initializing components",
		"[INFO] Components initialized successfully",
		"[WARN] Low disk space",
		"[ERROR] Failed to connect to database",
		"[INFO] Retrying connection...",
		"[INFO] Connected to database",
	})
	return &logPaneView{viewport: vp, active: false, focused: false}
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
