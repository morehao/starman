package components

import (
	"strings"
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
	var parts []string
	if m.message != "" {
		switch m.msgLevel {
		case types.LevelError:
			parts = append(parts, m.theme.CardTitle.Foreground(m.theme.Error).Render(m.message))
		case types.LevelSuccess:
			parts = append(parts, m.theme.CardTitle.Foreground(m.theme.Success).Render(m.message))
		default:
			parts = append(parts, m.message)
		}
	}
	parts = append(parts, "q Quit  Ctrl+K 命令面板  Tab 切换面板  Ctrl+Enter 执行  ? 帮助")

	separator := "   "
	joined := strings.Join(parts, separator)

	return joined
}

