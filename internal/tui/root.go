package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Ruohao1/penta/internal/model"
	"github.com/Ruohao1/penta/internal/tui/components"
	viewpkg "github.com/Ruohao1/penta/internal/tui/views"
)

type root struct {
	width, height int

	commandModel components.CmdInputModel
	cmdOpen      bool

	views      map[pentaView]tea.Model
	activeView pentaView

	events <-chan model.Event
}

func Run() error {
	return RunWithEvents(nil)
}

func RunWithEvents(events <-chan model.Event) error {
	models := make(map[pentaView]tea.Model)
	models[DashboardView] = viewpkg.DashboardModel()

	initialModel := root{
		activeView:   DashboardView,
		commandModel: *components.NewCommandInputModel(),
		views:        models,
		events:       events,
	}
	p := tea.NewProgram(initialModel)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func (m root) Init() tea.Cmd {
	if m.events == nil {
		return m.views[m.activeView].Init()
	}
	return tea.Batch(
		m.views[m.activeView].Init(),
		waitForEvent(m.events),
	)
}

func (m root) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		_, _ = m.commandModel.Update(msg)
		for k, v := range m.views {
			v, _ = v.Update(msg)
			m.views[k] = v
		}
		return m, nil
	}

	if m.cmdOpen {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			switch msg.String() {
			case "enter":
				m.cmdOpen = false
				return m, m.commandModel.Submit()
			case "esc", "escape":
				m.cmdOpen = false
				return m, nil
			}
		}
		_, cmd = m.commandModel.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case ":":
			m.cmdOpen = true
			return m, nil
		}
	}
	m.views[m.activeView], cmd = m.views[m.activeView].Update(msg)

	if m.events != nil {
		if _, ok := msg.(model.Event); ok {
			return m, tea.Batch(cmd, waitForEvent(m.events))
		}
	}
	return m, cmd
}

func (m root) View() tea.View {
	base := m.views[m.activeView].View().Content

	if m.cmdOpen {
		modal := m.commandModel.View().Content
		mw, mh := lipgloss.Size(modal)

		x := max(0, (m.width-mw)/2)
		y := max(0, min(m.height*10/100, (m.height-mh)/2))

		base = lipgloss.NewCompositor(
			lipgloss.NewLayer(base).Z(0),
			lipgloss.NewLayer(modal).X(x).Y(y).Z(10),
		).Render()
	}

	return tea.View{Content: base, AltScreen: true}
}

func waitForEvent(events <-chan model.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return nil
		}
		return ev
	}
}
