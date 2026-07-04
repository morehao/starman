package searchinput

import (
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

type SearchExecutedMsg struct {
	Query string
}

type Model struct {
	query   string
	focused bool
	th      *theme.Theme
}

func NewModel() Model {
	t := theme.DefaultTheme()
	return Model{th: &t}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) IsFocused() bool { return m.focused }

const (
	enterKey     = 13
	escapeKey    = 27
	backspaceKey = 127
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.focused {
			return m, nil
		}
		switch msg.Code {
		case escapeKey:
			m.focused = false
			m.query = ""
			return m, nil
		case enterKey:
			m.focused = false
			q := m.query
			m.query = ""
			return m, func() tea.Msg { return SearchExecutedMsg{Query: q} }
		case backspaceKey:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
			}
		default:
			if msg.Code >= 32 && msg.Code < 127 {
				m.query += string(rune(msg.Code))
			}
		}
	case tea.WindowSizeMsg:
		return m, nil
	}
	return m, nil
}

func (m Model) View() tea.View {
	if !m.focused {
		return tea.NewView("")
	}
	return tea.NewView(
		lipgloss.NewStyle().Foreground(m.th.FaintText).Render("🔍 ") +
			lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.query) + "_",
	)
}
