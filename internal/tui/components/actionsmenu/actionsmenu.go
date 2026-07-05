package actionsmenu

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/theme"
)

type MenuItem struct {
	Action    string
	Label     string
	NeedRow   bool
	Separator bool
}

type ActionsMenuResultMsg struct {
	Action    string
	Confirmed bool
}

type Model struct {
	active  bool
	cursor  int
	hasRow  bool
	th      theme.Theme
	title   string
	items   []MenuItem
}

func New(title string, items []MenuItem, hasSelectedRow bool) Model {
	return Model{
		active:  true,
		cursor:  0,
		hasRow:  hasSelectedRow,
		title:   title,
		items:   items,
	}
}

func (m *Model) SetTheme(th theme.Theme) { m.th = th }
func (m Model) IsFocused() bool           { return m.active }
func (m Model) Init() tea.Cmd             { return nil }

func (m Model) visibleItems() []MenuItem {
	var r []MenuItem
	for _, it := range m.items {
		if it.NeedRow && !m.hasRow {
			continue
		}
		r = append(r, it)
	}
	return r
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.active {
			return m, nil
		}
		switch msg.Code {
		case 27: // escape
			m.active = false
			return m, func() tea.Msg { return ActionsMenuResultMsg{Confirmed: false} }
		case 13: // enter
			m.active = false
			items := m.visibleItems()
			if m.cursor >= 0 && m.cursor < len(items) {
				return m, func() tea.Msg {
					return ActionsMenuResultMsg{Action: items[m.cursor].Action, Confirmed: true}
				}
			}
			return m, func() tea.Msg { return ActionsMenuResultMsg{Confirmed: false} }
		case 106, tea.KeyDown: // j, down
			m.cursor++
			m.clampCursor()
		case 107, tea.KeyUp: // k, up
			m.cursor--
			m.clampCursor()
		}
	}
	return m, nil
}

func (m *Model) clampCursor() {
	items := m.visibleItems()
	if len(items) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(items) {
		m.cursor = 0
	} else if m.cursor < 0 {
		m.cursor = len(items) - 1
	}
}

func (m Model) View() tea.View {
	if !m.active {
		return tea.NewView("")
	}

	items := m.visibleItems()
	overlayWidth := 30

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.FaintBorder).
		Padding(1, 2).
		Width(overlayWidth)

	titleStyle := lipgloss.NewStyle().Foreground(m.th.PrimaryText).Bold(true).Width(overlayWidth - 4)
	dimStyle := lipgloss.NewStyle().Foreground(m.th.FaintText)
	selectStyle := lipgloss.NewStyle().Background(m.th.SelectedBackground).Foreground(m.th.PrimaryText).Width(overlayWidth - 4)

	var lines []string
	lines = append(lines, titleStyle.Render(m.title))
	lines = append(lines, "")

	for i, it := range items {
		if it.Separator {
			lines = append(lines, dimStyle.Render(it.Label))
		} else if i == m.cursor {
			lines = append(lines, selectStyle.Render("  "+it.Label))
		} else {
			lines = append(lines, dimStyle.Render("  "+it.Label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[Enter] 确认  [Esc] 取消"))

	var content string
	for _, line := range lines {
		content += line + "\n"
	}
	return tea.NewView(overlayStyle.Render(content))
}

func (m Model) BoxView() string {
	if !m.active {
		return ""
	}
	return m.renderContentOnly()
}

func (m Model) renderContentOnly() string {
	items := m.visibleItems()

	titleStyle := lipgloss.NewStyle().Foreground(m.th.PrimaryText).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(m.th.FaintText)
	selectStyle := lipgloss.NewStyle().Background(m.th.SelectedBackground).Foreground(m.th.PrimaryText)

	var lines []string
	lines = append(lines, titleStyle.Render(m.title))
	lines = append(lines, "")

	for i, it := range items {
		if it.Separator {
			lines = append(lines, dimStyle.Render(it.Label))
		} else if i == m.cursor {
			lines = append(lines, selectStyle.Render("  "+it.Label))
		} else {
			lines = append(lines, dimStyle.Render("  "+it.Label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("[Enter] 确认  [Esc] 取消"))

	var content string
	for _, line := range lines {
		content += line + "\n"
	}
	return content
}
