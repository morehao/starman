package listviewport

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
)

type Column struct {
	Title string
	Width int
}

type Row struct {
	Columns []string
}

type Model struct {
	viewport viewport.Model
	columns  []Column
	rows     []Row
	cursor   int
}

func New(cols []Column) Model {
	return Model{viewport: viewport.New(), columns: cols}
}

func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	if m.cursor >= len(rows) && len(rows) > 0 {
		m.cursor = len(rows) - 1
	}
	m.viewport.SetContent(m.render())
}

func (m *Model) Cursor() int { return m.cursor }

func (m *Model) NextRow() {
	if m.cursor < len(m.rows)-1 {
		m.cursor++
		m.viewport.SetContent(m.render())
	}
}

func (m *Model) PrevRow() {
	if m.cursor > 0 {
		m.cursor--
		m.viewport.SetContent(m.render())
	}
}

func (m *Model) FirstItem() {
	m.cursor = 0
	m.viewport.SetContent(m.render())
}

func (m *Model) LastItem() {
	if len(m.rows) > 0 {
		m.cursor = len(m.rows) - 1
		m.viewport.SetContent(m.render())
	}
}

func (m Model) View() string { return m.viewport.View() }

func (m Model) Pager() string { return fmt.Sprintf("%d/%d", m.cursor+1, len(m.rows)) }

func (m Model) render() string {
	var b strings.Builder
	for i, r := range m.rows {
		prefix := " "
		if i == m.cursor {
			prefix = ">"
		}
		b.WriteString(prefix + " " + strings.Join(r.Columns, " "))
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return lipgloss.NewStyle().Render(b.String())
}
