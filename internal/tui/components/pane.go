package components

import tea "charm.land/bubbletea/v2"

type Pane interface {
	Init() tea.Cmd
	Update(tea.Msg) (Pane, tea.Cmd)
	View() tea.View

	Size() (int, int)
	SetSize(w, h int)
	Help() string 

	Focus() 
	Unfocus()
	IsFocused() bool
}
