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
	viewport    viewport.Model
	columns     []Column
	rows        []section.RowData
	cursor      int
	context     *context.ProgramContext
	width       int
	height      int
}

func New(cols []Column, programContext *context.ProgramContext) Model {
	return Model{
		viewport: viewport.New(),
		columns:  cols,
		context:  programContext,
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
	m.viewport.SetContent(m.render())
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
	if len(m.rows) > 0 {
		m.cursor = 0
		m.viewport.SetContent(m.render())
	}
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
	if len(m.columns) == 0 {
		return m.renderSimple()
	}
	return m.renderTable()
}

func (m Model) renderSimple() string {
	var b strings.Builder
	for i, r := range m.rows {
		prefix := " "
		if i == m.cursor {
			prefix = "▸"
		}
		b.WriteString(prefix + " " + r.GetTitle())
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) renderTable() string {
	if len(m.rows) == 0 {
		return ""
	}

	theme := m.context.Theme
	selectedBg := lipgloss.NewStyle().Background(theme.SelectedBackground)
	selectedFg := lipgloss.NewStyle().Foreground(theme.PrimaryText).Background(theme.SelectedBackground)
	faintStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	borderStyle := lipgloss.NewStyle().Foreground(theme.FaintBorder)

	var b strings.Builder

	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	for i, col := range m.columns {
		title := col.Title
		if len(title) > col.Width {
			title = title[:col.Width]
		}
		b.WriteString(fmt.Sprintf("%-*s", col.Width, title))
		if i < len(m.columns)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	for rowIdx, row := range m.rows {
		prefix := "  "
		if rowIdx == m.cursor {
			prefix = "▸ "
		}

		colVals := row.GetColumns()
		lineStr := prefix
		for colIdx, col := range m.columns {
			val := ""
			if colIdx < len(colVals) {
				val = colVals[colIdx]
			}
		if len(val) > col.Width {
			val = val[:col.Width-1] + "…"
		}
		lineStr += fmt.Sprintf("%-*s", col.Width, val)
			if colIdx < len(m.columns)-1 {
				lineStr += " "
			}
		}

		if rowIdx == m.cursor {
			if len(lineStr) > 2 {
				b.WriteString(selectedBg.Render(lineStr))
			} else {
				b.WriteString(selectedFg.Render(lineStr))
			}
		} else {
			b.WriteString(faintStyle.Render(lineStr))
		}
		if rowIdx < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
