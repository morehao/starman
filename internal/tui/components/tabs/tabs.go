package tabs

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type Model struct {
	ctx           *context.ProgramContext
	titles        []string
	sectionTabs   []string
	active        int
	activeSection int
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

func (m Model) Active() int {
	return m.active
}

func (m *Model) SetSectionTabs(tabs []string) {
	m.sectionTabs = tabs
}

func (m *Model) SetActiveSection(idx int) {
	if idx >= 0 && idx < len(m.sectionTabs) {
		m.activeSection = idx
	}
}

func (m *Model) NextSection() bool {
	if len(m.sectionTabs) == 0 {
		return false
	}
	m.activeSection = (m.activeSection + 1) % len(m.sectionTabs)
	return true
}

func (m *Model) PrevSection() bool {
	if len(m.sectionTabs) == 0 {
		return false
	}
	m.activeSection = (m.activeSection - 1 + len(m.sectionTabs)) % len(m.sectionTabs)
	return true
}

func (m Model) ActiveSectionIndex() int {
	return m.activeSection
}

func (m Model) HasSectionTabs() bool {
	return len(m.sectionTabs) > 0
}

func (m Model) Height() int {
	if len(m.sectionTabs) == 0 {
		return 1
	}
	return 3
}

func (m Model) ViewTabAtX(x int) int {
	offset := 0
	for i, t := range m.titles {
		w := lipgloss.Width(t) + 3
		if x >= offset && x < offset+w {
			return i
		}
		offset += w
	}
	return -1
}

func (m Model) SectionTabAtX(x int) int {
	offset := 0
	for i, t := range m.sectionTabs {
		w := lipgloss.Width(t) + 2
		if x >= offset && x < offset+w {
			return i
		}
		offset += w
	}
	return -1
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
	viewRow := bgStyle.Render(strings.Join(titleParts, " | "))

	if len(m.sectionTabs) == 0 {
		return viewRow
	}

	separatorStyle := lipgloss.NewStyle().Foreground(theme.FaintBorder).Width(m.ctx.ScreenWidth)
	sectionRow := m.renderSectionTabs()
	separator := separatorStyle.Render(strings.Repeat("─", m.ctx.ScreenWidth))
	return lipgloss.JoinVertical(lipgloss.Top, viewRow, separator, sectionRow)
}

func (m Model) renderSectionTabs() string {
	theme := m.ctx.Theme
	activeStyle := lipgloss.NewStyle().Background(theme.SelectedBackground).Foreground(theme.PrimaryText)
	faintStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	rowStyle := lipgloss.NewStyle().Width(m.ctx.ScreenWidth)

	var items []string
	for i, t := range m.sectionTabs {
		if i == m.activeSection {
			items = append(items, activeStyle.Render(" "+t+" "))
		} else {
			items = append(items, faintStyle.Render(" "+t+" "))
		}
	}
	return rowStyle.Render(strings.Join(items, "|"))
}
