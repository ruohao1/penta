package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	viewpkg "github.com/Ruohao1/penta/internal/tui/views"
)

type root struct {
	activeView pentaView

	views map[pentaView]tea.Model
}

func Run() error {
	models := make(map[pentaView]tea.Model)
	models[DashboardView] = viewpkg.DashboardModel()

	initialModel := root{DashboardView, models}
	p := tea.NewProgram(initialModel)
    if _, err := p.Run(); err != nil {
			return fmt.Errorf("Alas, there's been an error: %v", err)
    }
		return nil
}

func (m root) Init() tea.Cmd {
	return m.views[m.activeView].Init()
}

func (m root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
  }
	var cmd tea.Cmd
	m.views[m.activeView], cmd = m.views[m.activeView].Update(msg)
	return m, cmd
}

func (m root) View() tea.View {
	var v tea.View
	v.Content = m.views[m.activeView].View().Content
	v.AltScreen = true
	
	return v
}
