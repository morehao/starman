package sidebar

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type Model struct {
	ctx      *context.ProgramContext
	content  string
	viewport viewport.Model
	width    int
	height   int
}

func NewModel(ctx *context.ProgramContext) Model {
	vp := viewport.New()
	return Model{ctx: ctx, viewport: vp}
}

func (m *Model) SetContent(content string) {
	m.content = content
	m.viewport.SetContent(m.renderContent())
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	if w < 10 {
		w = 10
	}
	if h < 3 {
		h = 3
	}
	m.viewport.SetWidth(w)
	m.viewport.SetHeight(h)
	m.viewport.SetContent(m.renderContent())
}

func (m Model) View() string {
	w := m.ctx.DynamicPreviewWidth
	h := m.ctx.DynamicPreviewHeight
	if w < 1 {
		w = m.width
	}
	if h < 1 {
		h = m.height
	}
	if (m.ctx.DynamicPreviewWidth > 0 || m.width > 0) && (w < 20 || h < 10) {
		return ""
	}

	if m.content == "" {
		return lipgloss.NewStyle().
			Width(w).
			Height(h).
			Render("Nothing selected...")
	}

	return lipgloss.NewStyle().
		Width(w).
		Render(m.viewport.View())
}

func (m Model) renderContent() string {
	if m.content == "" {
		return ""
	}

	theme := m.ctx.Theme
	headerStyle := lipgloss.NewStyle().
		Background(theme.SelectedBackground).
		Foreground(theme.SecondaryText).
		PaddingLeft(1).
		Width(m.width - 1)

	parts := strings.SplitN(m.content, "\n", 2)
	header := parts[0]
	body := ""
	if len(parts) > 1 {
		body = parts[1]
	}

	var result strings.Builder
	result.WriteString(headerStyle.Render(header))
	result.WriteString("\n")
	result.WriteString(body)
	return result.String()
}
