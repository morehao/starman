package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type StatusBarModel struct {
	theme       *styles.Theme
	repoCount   int
	lastSync    string
	aiQuota     string
	message     string
	msgLevel    types.StatusLevel
	msgExpiry   time.Time
	taskSummary string
	width       int
}

func NewStatusBar(theme *styles.Theme) *StatusBarModel {
	return &StatusBarModel{
		theme:    theme,
		aiQuota:  "—",
		lastSync: "—",
	}
}

func (m *StatusBarModel) Init() tea.Cmd {
	return nil
}

func (m *StatusBarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case types.TickMsg:
		if time.Now().After(m.msgExpiry) && m.message != "" {
			m.message = ""
		}
	case types.StatusMsg:
		m.message = msg.Text
		m.msgLevel = msg.Level
		m.msgExpiry = time.Now().Add(msg.Timeout)
	}
	return m, nil
}

func (m *StatusBarModel) SetTaskSummary(summary string) {
	m.taskSummary = summary
}

func (m *StatusBarModel) View() string {
	left := fmt.Sprintf("⭐ %d repos | 🔄 %s", m.repoCount, m.lastSync)
	if m.taskSummary != "" {
		left += " | 📋 " + m.taskSummary
	}
	var mid string
	if m.message != "" {
		switch m.msgLevel {
		case types.LevelError:
			mid = m.theme.CardTitle.Foreground(m.theme.Error).Render(m.message)
		case types.LevelSuccess:
			mid = m.theme.CardTitle.Foreground(m.theme.Success).Render(m.message)
		default:
			mid = m.message
		}
	}
	right := "Ctrl+K:Palette  Tab:Switch  Ctrl+Enter:Run  Ctrl+C:Quit"

	leftW := len(left)
	midW := len(mid)
	rightW := len(right)
	available := m.width - leftW - rightW - 4
	if midW > available {
		mid = mid[:max(available-1, 0)]
	}
	padding := max(available-midW, 0) / 2

	return m.theme.StatusBar.Width(m.width).Render(
		left + repeat(" ", padding) + mid + repeat(" ", padding) + right,
	)
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
