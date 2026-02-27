package components

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var baseStyle = lipgloss.NewStyle().
	Padding(0, 0).
	Margin(0, 0)

type tablePane struct {
	table         table.Model
	width, height int
	columns       []Column

	styles table.Styles

	isFocused bool
}

type Column struct {
	Title        string
	WidthPercent int
}

func TablePane(columns []Column) *tablePane {
	var t table.Model

	tableColumns := make([]table.Column, len(columns))
	for i, col := range columns {
		tableColumns[i] = table.Column{
			Title: col.Title,
			Width: 0,
		}
	}

	rows := []table.Row{{"1", "New York", "USA", "8,398,748"}}

	t = table.New(
		table.WithColumns(tableColumns),
		table.WithRows(rows),
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Cell = s.Cell
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	s.Selected = s.Selected.Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Bold(true)

	t.SetStyles(s)

	return &tablePane{t, 0, 0, columns, s, false}
}

func (m tablePane) Init() tea.Cmd { return nil }

func (m *tablePane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func styleOverhead(s lipgloss.Style) int {
	// width of rendered 1-char cell minus 1 = overhead (padding + margins + borders in that style)
	return lipgloss.Width(s.Render("X")) - 1
}

func (p *tablePane) View() tea.View {
	p.table.SetHeight(p.height)
	p.table.SetWidth(p.width)

	styles := p.styles

	// Worst-case: selected rows can be widest (depends on your styles).
	cellOH := styleOverhead(styles.Cell)
	selOH  := styleOverhead(styles.Selected)
	hdrOH  := styleOverhead(styles.Header)

	perCellOH := max(cellOH, selOH, hdrOH)

	numCols := len(p.columns)

	// Table also inserts gaps between columns (commonly 1 space). Measure or assume 1.
	// If you want to be 100% sure, set it yourself (see note below).

	available := p.width - (numCols * perCellOH)
	available = max(1, available)

	// Now allocate column content widths based on 'available'
	cols := p.table.Columns()
	used := 0
	for i := 0; i < len(cols)-1; i++ {
		w := int(float64(available) * float64(p.columns[i].WidthPercent) / 100)
		w = max(1, w)
		cols[i].Width = w
		used += w
	}
	last := max(1, available-used)
	cols[len(cols)-1].Width = last
	p.table.SetColumns(cols)

	out := p.table.View()

	// Hard contract clamp: never exceed pane width/height even if table internals change
	out = lipgloss.NewStyle().
		Width(p.width).
		Height(p.height).
		MaxWidth(p.width).
		MaxHeight(p.height).
		Render(out)

	return tea.NewView(out)
}

func (m *tablePane) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *tablePane) SetRows(rows []table.Row) {
	m.table.SetRows(rows)
}

func (m *tablePane) AddRow(row table.Row) {
	rows := m.table.Rows()
	rows = append(rows, row)
	m.table.SetRows(rows)
}

func (m *tablePane) Help() string {
	return m.table.HelpView()
}

func (m *tablePane) Size() (int, int) {
	return m.width, m.height
}

func (m *tablePane) Focus() {
	m.isFocused = true
	m.table.Focus()
}

func (m *tablePane) Unfocus() {
	m.isFocused = false
	m.table.Blur()
}

func (m *tablePane) IsFocused() bool {
	return m.isFocused
}
