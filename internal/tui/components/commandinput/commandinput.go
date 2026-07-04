package commandinput

import (
	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/tui/components/inputoverlay"
	"github.com/morehao/starman/internal/tui/theme"
)

type CommandExecutedMsg struct {
	Command string
}

type Model struct {
	query      string
	focused    bool
	th         theme.Theme
	width      int
	height     int
	history    []string
	historyPos int
}

func NewModel() Model {
	return Model{
		th:         theme.DefaultTheme(),
		historyPos: -1,
	}
}

func (m *Model) SetTheme(th theme.Theme) {
	m.th = th
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) IsFocused() bool { return m.focused }

func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if focused {
		m.historyPos = -1
		m.query = ""
	}
}

const (
	enterKey     = 13
	escapeKey    = 27
	backspaceKey = 127
	upKey        = 65517
	downKey      = 65516
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
			m.historyPos = -1
			return m, nil
		case enterKey:
			m.focused = false
			cmd := m.query
			m.query = ""
			if cmd != "" {
				m.history = append(m.history, cmd)
			}
			m.historyPos = -1
			return m, func() tea.Msg { return CommandExecutedMsg{Command: cmd} }
		case upKey:
			if len(m.history) == 0 {
				return m, nil
			}
			if m.historyPos == -1 {
				m.historyPos = len(m.history) - 1
			} else if m.historyPos > 0 {
				m.historyPos--
			}
			m.query = m.history[m.historyPos]
			return m, nil
		case downKey:
			if m.historyPos == -1 {
				return m, nil
			}
			if m.historyPos < len(m.history)-1 {
				m.historyPos++
				m.query = m.history[m.historyPos]
			} else {
				m.historyPos = -1
				m.query = ""
			}
			return m, nil
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
	return inputoverlay.RenderOverlay(m.th, m.width, m.height, "Command", m.query)
}
