package prompt

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

type PromptType int

const (
	PromptConfirm PromptType = iota
	PromptCategorySelect
	PromptTagEdit
)

type PromptResultMsg struct {
	Type      PromptType
	Confirmed bool
	Value     string
}

type Model struct {
	ptype   PromptType
	title   string
	options []string
	input   string
	cursor  int
	active  bool
	th      theme.Theme
	width   int
}

func NewConfirmModel(title string) Model {
	return Model{ptype: PromptConfirm, title: title, active: true}
}

func NewCategorySelectModel(title string, categories []string, current string) Model {
	cursor := 0
	for i, c := range categories {
		if c == current {
			cursor = i
			break
		}
	}
	return Model{
		ptype:   PromptCategorySelect,
		title:   title,
		options: categories,
		cursor:  cursor,
		active:  true,
	}
}

func NewTagEditModel(title string, current string) Model {
	return Model{ptype: PromptTagEdit, title: title, input: current, active: true}
}

func (m *Model) SetTheme(th theme.Theme) {
	m.th = th
}

func (m Model) IsFocused() bool { return m.active }

func (m Model) Init() tea.Cmd { return nil }

const (
	enterKey     = 13
	escapeKey    = 27
	backspaceKey = 127
	jKey         = 106
	kKey         = 107
	yKey         = 121
	nKey         = 110
	YKey         = 89
	NKey         = 78
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.active {
			return m, nil
		}
		switch m.ptype {
		case PromptConfirm:
			return m.handleConfirmKeys(msg)
		case PromptCategorySelect:
			return m.handleCategoryKeys(msg)
		case PromptTagEdit:
			return m.handleTagKeys(msg)
		}
	case tea.WindowSizeMsg:
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case escapeKey:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptConfirm, Confirmed: false} }
	case yKey, YKey:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptConfirm, Confirmed: true} }
	case nKey, NKey:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptConfirm, Confirmed: false} }
	}
	return m, nil
}

func (m Model) handleCategoryKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case escapeKey:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptCategorySelect, Confirmed: false} }
	case enterKey:
		m.active = false
		return m, func() tea.Msg {
			return PromptResultMsg{Type: PromptCategorySelect, Confirmed: true, Value: m.options[m.cursor]}
		}
	case tea.KeyDown:
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
	case jKey:
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case kKey:
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m, nil
}

func (m Model) handleTagKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case escapeKey:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptTagEdit, Confirmed: false} }
	case enterKey:
		m.active = false
		return m, func() tea.Msg {
			return PromptResultMsg{Type: PromptTagEdit, Confirmed: true, Value: strings.TrimSpace(m.input)}
		}
	case backspaceKey:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if msg.Code >= 32 && msg.Code < 127 {
			m.input += string(rune(msg.Code))
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if !m.active {
		return tea.NewView("")
	}

	overlayWidth := max(30, m.width/2)

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.FaintBorder).
		Padding(1, 2).
		Width(overlayWidth)

	var content string
	switch m.ptype {
	case PromptConfirm:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.title) + "\n" +
			lipgloss.NewStyle().Foreground(m.th.FaintText).Render("[y] yes  [n] no")
	case PromptCategorySelect:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.title) + "\n"
		for i, opt := range m.options {
			prefix := "  "
			line := prefix + opt
			if i == m.cursor {
				prefix = "> "
				line = prefix + opt
				content += lipgloss.NewStyle().
					Background(m.th.SelectedBackground).
					Foreground(m.th.PrimaryText).
					Render(line)
			} else {
				content += lipgloss.NewStyle().Foreground(m.th.FaintText).Render(line)
			}
			content += "\n"
		}
	case PromptTagEdit:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.title) + "\n" +
			lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.input) + "_"
	}

	return tea.NewView(overlayStyle.Render(content))
}
