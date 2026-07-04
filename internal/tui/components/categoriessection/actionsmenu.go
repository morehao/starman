package categoriessection

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

const (
	ActionAdd    = "add"
	ActionEdit   = "edit"
	ActionDelete = "delete"
)

type ActionsMenuResultMsg struct {
	Action    string
	Confirmed bool
}

type ActionsMenuModel struct {
	active  bool
	cursor  int
	hasRow  bool
	th      theme.Theme
	options []actionOption
}

type actionOption struct {
	action  string
	label   string
	needRow bool
}

func NewActionsMenuModel(hasSelectedRow bool) ActionsMenuModel {
	opts := []actionOption{
		{action: ActionAdd, label: "+ 新增分类", needRow: false},
		{action: ActionEdit, label: "✏ 编辑当前分类", needRow: true},
		{action: ActionDelete, label: "✕ 删除当前分类", needRow: true},
	}
	return ActionsMenuModel{
		active:  true,
		cursor:  0,
		hasRow:  hasSelectedRow,
		options: opts,
	}
}

func (m *ActionsMenuModel) SetTheme(th theme.Theme) { m.th = th }
func (m ActionsMenuModel) IsFocused() bool           { return m.active }

func (m ActionsMenuModel) Init() tea.Cmd { return nil }

func (m ActionsMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			action := m.options[m.cursor].action
			return m, func() tea.Msg { return ActionsMenuResultMsg{Action: action, Confirmed: true} }
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

func (m *ActionsMenuModel) clampCursor() {
	if m.hasRow {
		if m.cursor >= len(m.options) {
			m.cursor = 0
		} else if m.cursor < 0 {
			m.cursor = len(m.options) - 1
		}
	} else {
		m.cursor = 0
	}
}

func (m ActionsMenuModel) View() tea.View {
	if !m.active {
		return tea.NewView("")
	}

	overlayWidth := 28
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.FaintBorder).
		Padding(1, 2).
		Width(overlayWidth)

	titleStyle := lipgloss.NewStyle().Foreground(m.th.PrimaryText).Bold(true).Width(overlayWidth - 4)
	dimStyle := lipgloss.NewStyle().Foreground(m.th.FaintText)
	selectStyle := lipgloss.NewStyle().Background(m.th.SelectedBackground).Foreground(m.th.PrimaryText).Width(overlayWidth - 4)
	helpStyle := lipgloss.NewStyle().Foreground(m.th.FaintText)

	var lines []string
	lines = append(lines, titleStyle.Render("Category 操作"))
	lines = append(lines, "")

	addOpt := m.options[0]
	if m.cursor == 0 {
		lines = append(lines, selectStyle.Render("  "+addOpt.label))
	} else {
		lines = append(lines, dimStyle.Render("  "+addOpt.label))
	}
	for i := 1; i < len(m.options); i++ {
		opt := m.options[i]
		if opt.needRow && !m.hasRow {
			continue
		}
		if i == m.cursor {
			lines = append(lines, selectStyle.Render("  "+opt.label))
		} else {
			lines = append(lines, dimStyle.Render("  "+opt.label))
		}
	}

	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("[Enter] 确认  [Esc] 取消"))

	content := ""
	for _, line := range lines {
		content += line + "\n"
	}
	return tea.NewView(overlayStyle.Render(content))
}
