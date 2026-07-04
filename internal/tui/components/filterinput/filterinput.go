package filterinput

import (
	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/tui/components/inputoverlay"
	"github.com/morehao/starman/internal/tui/theme"
)

type FilterExecutedMsg struct {
	Query string
}

type Model struct {
	query       string
	focused     bool
	th          theme.Theme
	width       int
	height      int
	sectionName string
}

func NewModel() Model {
	return Model{
		th: theme.DefaultTheme(),
	}
}

func (m *Model) SetTheme(th theme.Theme) {
	m.th = th
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Model) SetSectionName(name string) {
	m.sectionName = name
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) IsFocused() bool { return m.focused }

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.query = ""
	}
}

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
			return m, func() tea.Msg { return FilterExecutedMsg{Query: q} }
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

	return tea.NewView(m.renderOverlay())
}

func (m Model) renderOverlay() string {
	title := "Filter " + m.sectionName
	return inputoverlay.RenderOverlay(m.th, m.width, m.height, title, m.query)
}
