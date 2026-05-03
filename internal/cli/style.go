package cli

import "github.com/charmbracelet/lipgloss"

type cliStyles struct {
	heading  lipgloss.Style
	success  lipgloss.Style
	failure  lipgloss.Style
	phase    lipgloss.Style
	queued   lipgloss.Style
	running  lipgloss.Style
	evidence lipgloss.Style
	done     lipgloss.Style
	failed   lipgloss.Style
	debug    lipgloss.Style
}

func newCLIStyles(enabled bool) cliStyles {
	if !enabled {
		return cliStyles{}
	}
	return cliStyles{
		heading:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")),
		success:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
		failure:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),
		phase:    lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		queued:   lipgloss.NewStyle().Faint(true),
		running:  lipgloss.NewStyle().Foreground(lipgloss.Color("14")),
		evidence: lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		done:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")),
		failed:   lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
		debug:    lipgloss.NewStyle().Faint(true),
	}
}

func (s cliStyles) label(label string) string {
	switch label {
	case "queued":
		return s.queued.Render(label)
	case "running":
		return s.running.Render(label)
	case "evidence":
		return s.evidence.Render(label)
	case "done":
		return s.done.Render(label)
	case "failed":
		return s.failed.Render(label)
	default:
		return label
	}
}
