package commandmode

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

type CommandExecutedMsg struct {
	Input string
}

type Model struct {
	input   string
	focused bool
	th      theme.Theme
	width   int
	height  int
}

func NewModel() Model {
	return Model{th: theme.DefaultTheme()}
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
	if !focused {
		m.input = ""
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
			m.input = ""
			return m, nil
		case enterKey:
			m.focused = false
			in := m.input
			m.input = ""
			return m, func() tea.Msg { return CommandExecutedMsg{Input: in} }
		case backspaceKey:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if msg.Code >= 32 && msg.Code < 127 {
				m.input += string(rune(msg.Code))
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

func (m Model) BoxView() string {
	if !m.focused {
		return ""
	}
	return m.renderContentOnly()
}

func (m Model) renderContentOnly() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(m.th.PrimaryText).
		Bold(true)

	inputLabelStyle := lipgloss.NewStyle().
		Foreground(m.th.FaintText)

	inputStyle := lipgloss.NewStyle().
		Foreground(m.th.PrimaryText)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.th.FaintText)

	inputDisplay := m.input
	if inputDisplay == "" {
		inputDisplay = " "
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Execute Command"))
	b.WriteByte('\n')
	b.WriteString(inputLabelStyle.Render("$ "))
	b.WriteString(inputStyle.Render(inputDisplay + "█"))
	b.WriteByte('\n')
	b.WriteString(hintStyle.Render("Enter to execute  Esc to cancel"))

	return b.String()
}

const overlayWidth = 42

func (m Model) renderOverlay() string {
	dialogWidth := overlayWidth
	if m.width > 0 && m.width < dialogWidth+4 {
		dialogWidth = m.width - 4
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.FaintBorder).
		Padding(1, 2).
		Width(dialogWidth)

	titleStyle := lipgloss.NewStyle().
		Foreground(m.th.PrimaryText).
		Bold(true)

	inputLabelStyle := lipgloss.NewStyle().
		Foreground(m.th.FaintText)

	inputStyle := lipgloss.NewStyle().
		Foreground(m.th.PrimaryText)

	hintStyle := lipgloss.NewStyle().
		Foreground(m.th.FaintText)

	contentWidth := dialogWidth - 6
	if contentWidth < 0 {
		contentWidth = 0
	}
	separator := ""
	if contentWidth > 0 {
		separator = lipgloss.NewStyle().
			Foreground(m.th.FaintBorder).
			Render(strings.Repeat("─", contentWidth))
	}

	inputDisplay := m.input
	if inputDisplay == "" {
		inputDisplay = " "
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Execute Command"))
	b.WriteByte('\n')
	if separator != "" {
		b.WriteString(separator)
		b.WriteByte('\n')
	}
	b.WriteString(inputLabelStyle.Render("$ "))
	b.WriteString(inputStyle.Render(inputDisplay + "█"))
	b.WriteByte('\n')
	b.WriteString(hintStyle.Render("Enter to execute  Esc to cancel"))

	rendered := dialogStyle.Render(b.String())

	if m.width == 0 || m.height == 0 {
		return rendered
	}

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		rendered,
	)
}
