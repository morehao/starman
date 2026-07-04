package drawer

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

type entry struct {
	timestamp time.Time
	command   string
	stdout    string
	stderr    string
}

type Model struct {
	entries []entry
	open    bool
	focused bool
	width   int
	height  int
	vp      viewport.Model
	th      theme.Theme
}

func NewModel() Model {
	th := theme.DefaultTheme()
	vp := viewport.New()
	return Model{
		th: th,
		vp: vp,
	}
}

func (m *Model) SetTheme(th theme.Theme) {
	m.th = th
}

func (m *Model) AddEntry(command, stdout, stderr string) {
	m.entries = append(m.entries, entry{
		timestamp: time.Now(),
		command:   command,
		stdout:    stdout,
		stderr:    stderr,
	})
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	vpWidth := width - 2
	vpHeight := height - 2
	if vpWidth < 1 {
		vpWidth = 1
	}
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.vp.SetWidth(vpWidth)
	m.vp.SetHeight(vpHeight)
}

func (m Model) IsOpen() bool    { return m.open }
func (m Model) IsFocused() bool { return m.focused }

func (m *Model) SetOpen(open bool) {
	m.open = open
	m.focused = open
}

func (m Model) Init() tea.Cmd { return nil }

const (
	ctrlO = 15 // Ctrl+O
	ctrlL = 12 // Ctrl+L
	esc   = 27 // Escape
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.Code {
		case ctrlO:
			m.open = !m.open
			m.focused = m.open
			return m, nil
		case esc:
			if m.open {
				m.open = false
				m.focused = false
				return m, nil
			}
		case ctrlL:
			m.entries = nil
			return m, nil
		}
		if m.open {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if !m.open || len(m.entries) == 0 {
		return tea.NewView("")
	}

	var b strings.Builder
	for i := len(m.entries) - 1; i >= 0; i-- {
		e := m.entries[i]
		b.WriteString(lipgloss.NewStyle().Foreground(m.th.FaintText).Render(e.timestamp.Format("15:04:05") + " "))
		b.WriteString(lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(e.command))
		b.WriteString("\n")
		if e.stdout != "" {
			b.WriteString(e.stdout)
		}
		if e.stderr != "" {
			b.WriteString(lipgloss.NewStyle().Foreground(m.th.ErrorText).Render(e.stderr))
		}
		if i > 0 {
			b.WriteString("\n")
		}
	}

	m.vp.SetContent(b.String())

	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.th.FaintBorder).
		Width(m.width)
	return tea.NewView(border.Render(m.vp.View()))
}
