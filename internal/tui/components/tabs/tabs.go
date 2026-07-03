package tabs

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type Model struct {
	ctx    *context.ProgramContext
	titles []string
	active int
}

func NewModel(ctx *context.ProgramContext) Model {
	return Model{ctx: ctx}
}

func (m *Model) SetTitles(titles []string) {
	m.titles = titles
}

func (m *Model) SetActive(active int) {
	m.active = active
}

func (m Model) View() string {
	theme := m.ctx.Theme
	faintStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	activeStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)
	bgStyle := lipgloss.NewStyle().Background(theme.SelectedBackground).Width(m.ctx.ScreenWidth)

	var titleParts []string
	for i, t := range m.titles {
		if i == m.active {
			titleParts = append(titleParts, activeStyle.Render(t))
		} else {
			titleParts = append(titleParts, faintStyle.Render(t))
		}
	}
	return bgStyle.Render(strings.Join(titleParts, " | "))
}
