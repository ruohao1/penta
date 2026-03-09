package components

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ruohao1/penta/internal/tui/styles"
)

var cmdInputStyle = lipgloss.NewStyle().BorderForeground(styles.ColorPaneBorderActive).Border(lipgloss.NormalBorder()).Padding(0, 1)

type CmdInputModel struct {
	textInput textinput.Model

	screenWidth, screenHeight int
}

func NewCommandInputModel() *CmdInputModel {
	ti := textinput.New()
	ti.Placeholder = "Command"
	ti.CharLimit = 156
	ti.SetVirtualCursor(false)
	ti.Focus()

	return &CmdInputModel{
		textInput: ti,
	}
}

func (m *CmdInputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *CmdInputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenWidth, m.screenHeight = msg.Width, msg.Height
		m.textInput.SetWidth(m.screenWidth * 40 / 100)
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *CmdInputModel) View() tea.View {
	var c *tea.Cursor
	if !m.textInput.VirtualCursor() {
		c = m.textInput.Cursor()
	}
	inputWidth := m.screenWidth * 40 / 100
	str := styles.WithTitle(cmdInputStyle, "Command", m.textInput.View(), inputWidth, 1)

	v := tea.NewView(str)
	v.Cursor = c
	return v
}

func (m *CmdInputModel) Submit() tea.Cmd {
	m.textInput.Reset()
	return nil
}

func (m *CmdInputModel) Reset() {
	m.textInput.Reset()
}
