package listviewport

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/context"
)

type Column struct {
	Title string
	Width int
}

type Model struct {
	viewport viewport.Model
	columns  []Column
	rows     []section.RowData
	cursor   int
	context  *context.ProgramContext
}

func New(cols []Column, programContext *context.ProgramContext) Model {
	return Model{viewport: viewport.New(), columns: cols, context: programContext}
}

func (m *Model) SetRows(rows []section.RowData) {
	m.rows = rows
	if len(rows) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(rows) {
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
	if len(m.rows) == 0 {
		m.cursor = 0
	} else {
		m.cursor = 0
	}
	m.viewport.SetContent(m.render())
}

func (m *Model) LastItem() {
	if len(m.rows) > 0 {
		m.cursor = len(m.rows) - 1
		m.viewport.SetContent(m.render())
	}
}

func (m Model) View() string { return m.viewport.View() }

func (m Model) Pager() string {
	if len(m.rows) == 0 {
		return "0/0"
	}
	return fmt.Sprintf("%d/%d", m.cursor+1, len(m.rows))
}

func (m Model) render() string {
	var b strings.Builder
	for i, r := range m.rows {
		prefix := " "
		if i == m.cursor {
			prefix = ">"
		}
		b.WriteString(prefix + " " + r.GetTitle())
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return lipgloss.NewStyle().Render(b.String())
}
