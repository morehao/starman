package searchinput

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

type SearchExecutedMsg struct {
	Query string
}

type Model struct {
	query     string
	cursorPos int
	focused   bool
	th        theme.Theme
	width     int
	height    int
	viewType  string
}

const (
	ViewTypeStars      = "Stars"
	ViewTypeCategories = "Categories"
)

func NewModel() Model {
	return Model{th: theme.DefaultTheme()}
}

func (m *Model) SetTheme(th theme.Theme) {
	m.th = th
}

func (m *Model) SetViewType(vt string) {
	m.viewType = vt
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
		m.query = ""
		m.cursorPos = 0
	} else {
		m.cursorPos = len(m.query)
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
			m.cursorPos = 0
			return m, nil
		case enterKey:
			m.focused = false
			q := m.query
			m.query = ""
			m.cursorPos = 0
			return m, func() tea.Msg { return SearchExecutedMsg{Query: q} }
		case backspaceKey:
			if m.cursorPos > 0 {
				m.query = m.query[:m.cursorPos-1] + m.query[m.cursorPos:]
				m.cursorPos--
			}
		case tea.KeyDelete:
			if m.cursorPos < len(m.query) {
				m.query = m.query[:m.cursorPos] + m.query[m.cursorPos+1:]
			}
		case tea.KeyLeft:
			if m.cursorPos > 0 {
				m.cursorPos--
			}
		case tea.KeyRight:
			if m.cursorPos < len(m.query) {
				m.cursorPos++
			}
		case tea.KeyHome:
			m.cursorPos = 0
		case tea.KeyEnd:
			m.cursorPos = len(m.query)
		default:
			if msg.Code >= 32 && msg.Code < 127 {
				m.query = m.query[:m.cursorPos] + string(rune(msg.Code)) + m.query[m.cursorPos:]
				m.cursorPos++
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
	return m.renderDialogBox()
}

const overlayWidth = 42

func (m Model) renderOverlay() string {
	rendered := m.renderDialogBox()

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

func (m Model) renderDialogBox() string {
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

	queryDisplay := m.query

	pos := m.cursorPos
	if pos > len(queryDisplay) {
		pos = len(queryDisplay)
	}

	var b strings.Builder
	title := "Search " + m.viewType
	if m.viewType == "" {
		title = "Search"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteByte('\n')
	if separator != "" {
		b.WriteString(separator)
		b.WriteByte('\n')
	}
	b.WriteString(inputLabelStyle.Render("🔍 "))
	b.WriteString(inputStyle.Render(queryDisplay[:pos] + "█" + queryDisplay[pos:]))
	b.WriteByte('\n')
	b.WriteString(hintStyle.Render("Enter to search  Esc to cancel"))

	return dialogStyle.Render(b.String())
}
