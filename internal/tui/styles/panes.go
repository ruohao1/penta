package styles

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var DefaultPaneStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(ColorPaneBorderDefault).
	Padding(0, 0)

var ActivePaneStyle = DefaultPaneStyle.BorderForeground(ColorPaneBorderActive)

var FocusedPaneStyle = ActivePaneStyle.BorderForeground(ColorPaneBorderFocused)

func WithTitle(base lipgloss.Style, title, content string, w, h int) string {
	b := base.GetBorderStyle()

	topLine := b.TopLeft + " " + title + " " +
		strings.Repeat("─", max(0, w-lipgloss.Width(title)-4)) + b.TopRight

	top := lipgloss.NewStyle().
		Foreground(base.GetBorderTopForeground()).
		Render(topLine)

	body := base.
		Border(b, false, true, true, true).
		Width(w).
		Height(max(1, h-1)).
		Render(content)

	return lipgloss.JoinVertical(lipgloss.Left, top, body)
}
