package listviewport

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"

	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/context"
)

type Column struct {
	Title string
	Width int
	Flex  bool
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
	if m.width == w && m.height == h {
		return
	}
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

const tableHeaderLines = 3

func (m *Model) scrollToCursor() {
	if len(m.rows) == 0 || m.height <= 0 {
		return
	}
	cursorLine := tableHeaderLines + m.cursor
	y := m.viewport.YOffset()
	h := m.height

	if cursorLine >= y && cursorLine < y+h {
		return
	}

	if cursorLine < y {
		m.viewport.SetYOffset(0)
	} else {
		m.viewport.SetYOffset(cursorLine - h + 1)
	}
}

func (m *Model) NextRow() {
	if m.cursor < len(m.rows)-1 {
		m.cursor++
		m.viewport.SetContent(m.render())
		m.scrollToCursor()
	}
}

func (m *Model) PrevRow() {
	if m.cursor > 0 {
		m.cursor--
		m.viewport.SetContent(m.render())
		m.scrollToCursor()
	}
}

func (m *Model) FirstItem() {
	if len(m.rows) > 0 {
		m.cursor = 0
		m.viewport.SetContent(m.render())
		m.scrollToCursor()
	}
}

func (m *Model) LastItem() {
	if len(m.rows) > 0 {
		m.cursor = len(m.rows) - 1
		m.viewport.SetContent(m.render())
		m.scrollToCursor()
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

	cols, contentW := m.fitColumns(m.width)
	if contentW < 1 {
		return ""
	}

	theme := m.context.Theme
	selectedBg := lipgloss.NewStyle().Background(theme.SelectedBackground)
	selectedFg := lipgloss.NewStyle().Foreground(theme.PrimaryText).Background(theme.SelectedBackground)
	faintStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	borderStyle := lipgloss.NewStyle().Foreground(theme.FaintBorder)

	var b strings.Builder

	b.WriteString(borderStyle.Render(strings.Repeat("─", contentW)))
	b.WriteString("\n")

	for i, col := range cols {
		title := col.Title
		if runewidth.StringWidth(title) > col.Width {
			title = runewidth.Truncate(title, col.Width, "…")
		}
		b.WriteString(padRight(title, col.Width))
		if i < len(cols)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", contentW)))
	b.WriteString("\n")

	for rowIdx, row := range m.rows {
		prefix := "  "
		if rowIdx == m.cursor {
			prefix = "▸ "
		}

		colVals := row.GetColumns()
		var lineStr string
		if len(colVals) == 0 {
			title := row.GetTitle()
			avail := contentW - runewidth.StringWidth(prefix)
			if avail < 4 {
				lineStr = prefix + title
			} else {
				inner := "── " + title + " ──"
				if runewidth.StringWidth(inner) > avail {
					inner = "── " + title
					if runewidth.StringWidth(inner) > avail-1 {
						inner = title
						if runewidth.StringWidth(inner) > avail {
							inner = runewidth.Truncate(inner, avail, "")
						}
					} else {
						inner += " ─"
					}
				}
				fill := avail - runewidth.StringWidth(inner)
				if fill > 0 {
					lineStr = prefix + inner + strings.Repeat("─", fill)
				} else {
					lineStr = prefix + inner
				}
			}
			b.WriteString(borderStyle.Render(lineStr))
		} else {
			lineStr = prefix
			for colIdx, col := range cols {
				val := ""
				if colIdx < len(colVals) {
					val = colVals[colIdx]
				}
				if runewidth.StringWidth(val) > col.Width {
					val = runewidth.Truncate(val, col.Width-1, "…")
				}
				lineStr += padRight(val, col.Width)
				if colIdx < len(cols)-1 {
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
		}
		if rowIdx < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) fitColumns(availWidth int) ([]Column, int) {
	n := len(m.columns)
	if availWidth <= 0 || n == 0 {
		return m.columns, 0
	}
	gap := n - 1
	prefixW := 2
	contentAvail := availWidth - prefixW - gap
	if contentAvail < n*3 {
		contentAvail = n * 3
	}

	idealContent := 0
	flexIdeal := 0
	for _, c := range m.columns {
		idealContent += c.Width
		if c.Flex {
			flexIdeal += c.Width
		}
	}

	fitted := make([]Column, n)
	copy(fitted, m.columns)

	if idealContent <= contentAvail {
		extra := contentAvail - idealContent
		if extra > 0 && flexIdeal > 0 {
			distributed := 0
			for i := range fitted {
				if m.columns[i].Flex {
					share := extra * m.columns[i].Width / flexIdeal
					fitted[i].Width += share
					distributed += share
				}
			}
			remainder := extra - distributed
			if remainder > 0 {
				for i := n - 1; i >= 0; i-- {
					if m.columns[i].Flex {
						fitted[i].Width += remainder
						break
					}
				}
			}
		}
	} else {
		scale := float64(contentAvail) / float64(idealContent)
		if scale < 0.3 {
			scale = 0.3
		}
		for i := range fitted {
			w := int(float64(m.columns[i].Width) * scale)
			if w < 3 {
				w = 3
			}
			fitted[i].Width = w
		}
	}

	used := prefixW
	for i := range fitted {
		used += fitted[i].Width
		if i < n-1 {
			used++
		}
	}
	return fitted, used
}

func padRight(s string, width int) string {
	dw := runewidth.StringWidth(s)
	if dw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-dw)
}
