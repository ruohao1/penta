package components

import (
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ruohao1/penta/internal/tui/messages"
	"github.com/ruohao1/penta/internal/tui/styles"
)

type tablePane struct {
	table   table.Model
	columns []Column

	styles table.Styles

	width, height int
	active        bool
	focused       bool
}

type Column struct {
	Title        string
	WidthPercent int
}

type TableRows interface {
	AddStringRow(row []string)
	SetStringRows(rows [][]string)
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
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderBottom(true)
	s.Selected = s.Selected.Background(styles.ColorSurfaceAccent).Foreground(styles.ColorTextPrimary).Bold(true)

	t.SetStyles(s)

	return &tablePane{
		table:   t,
		columns: columns,
		styles:  s,
		active:  false,
		focused: false,
	}
}

func (m tablePane) Init() tea.Cmd { return nil }

func (m *tablePane) Update(msg tea.Msg) (Pane, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case messages.PaneSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
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
	return lipgloss.Width(s.Render("X")) - 1
}

func (p *tablePane) View() tea.View {
	out := p.table.View()
	var paneStyle lipgloss.Style
	switch {
	case p.focused:
		paneStyle = styles.FocusedPaneStyle
	case p.active:
		paneStyle = styles.ActivePaneStyle
	default:
		paneStyle = styles.DefaultPaneStyle
	}

	out = styles.WithTitle(paneStyle, "Table", out, p.width, p.height)

	return tea.NewView(out)
}

func (m *tablePane) SetSize(width, height int) {
	m.width = width
	m.height = height

	stylesTbl := m.styles

	cellOH := styleOverhead(stylesTbl.Cell)
	hdrOH := styleOverhead(stylesTbl.Header)
	selOH := styleOverhead(stylesTbl.Selected)
	perCellOH := max(cellOH, max(hdrOH, selOH))

	numCols := len(m.columns)
	colGap := max(0, numCols-1)
	available := m.width - (numCols * perCellOH) - colGap
	available = max(1, available)

	cols := m.table.Columns()
	used := 0
	for i := 0; i < len(cols)-1; i++ {
		w := max(1, available*m.columns[i].WidthPercent/100)
		cols[i].Width = w
		used += w
	}
	if len(cols) > 0 {
		cols[len(cols)-1].Width = max(1, available-used)
	}
	m.table.SetColumns(cols)
	m.table.SetHeight(max(1, m.height-2)) // Account for borders.
	m.table.SetWidth(max(1, m.width-2))   // Account for borders
}

func (m *tablePane) SetRows(rows []table.Row) {
	m.table.SetRows(rows)
}

func (m *tablePane) AddRow(row table.Row) {
	rows := m.table.Rows()
	rows = append(rows, row)
	m.table.SetRows(rows)
}

func (m *tablePane) AddStringRow(row []string) {
	m.AddRow(table.Row(row))
}

func (m *tablePane) SetStringRows(rows [][]string) {
	out := make([]table.Row, 0, len(rows))
	for _, r := range rows {
		out = append(out, table.Row(r))
	}
	m.SetRows(out)
}

func (m *tablePane) Help() string {
	return m.table.HelpView()
}

func (m *tablePane) Size() (int, int) {
	return m.width, m.height
}

func (m *tablePane) Focused() bool {
	return m.focused
}

func (m *tablePane) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.table.Focus()
	} else {
		m.table.Blur()
	}
}

func (m *tablePane) Active() bool {
	return m.active
}

func (m *tablePane) SetActive(active bool) {
	m.active = active
}
