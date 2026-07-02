package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type SyncModel struct {
	store  store.Store
	theme  *styles.Theme
	stats  *store.SyncStats
	width  int
	height int
	loaded bool
	syncing bool
}

type syncStatsMsg struct{ stats *store.SyncStats }
type syncDoneMsg struct{ stats *store.SyncStats; err error }

func NewSync(s store.Store, theme *styles.Theme) *SyncModel {
	return &SyncModel{store: s, theme: theme}
}

func (m *SyncModel) Init() tea.Cmd { return m.loadStatsCmd }

func (m *SyncModel) loadStatsCmd() tea.Msg {
	ctx := context.Background()
	stats, err := m.store.GetSyncStats(ctx)
	if err != nil {
		return syncStatsMsg{stats: &store.SyncStats{}}
	}
	return syncStatsMsg{stats: stats}
}

func (m *SyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case syncStatsMsg:
		m.stats = msg.stats
		m.loaded = true
		m.syncing = false
	case syncDoneMsg:
		m.stats = msg.stats
		m.syncing = false
		if msg.err != nil {
			return m, nil
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			if !m.syncing {
				m.syncing = true
				return m, func() tea.Msg {
					return syncDoneMsg{stats: m.stats}
				}
			}
		case "esc":
			return m, func() tea.Msg { return types.NavigatedMsg{Page: types.PageDashboard} }
		}
	}
	return m, nil
}

func (m *SyncModel) View() string {
	if !m.loaded {
		return "loading sync info..."
	}
	title := m.theme.PageTitle.Render("Sync") + "\n\n"

	lastSync := "Never"
	if !m.stats.LastSync.IsZero() {
		lastSync = m.stats.LastSync.Format("2006-01-02 15:04:05")
	}
	duration := m.stats.LastDuration
	if duration == "" {
		duration = "—"
	}

	cards := []string{
		m.theme.Card.Render(fmt.Sprintf("🔄\n %d\n 总同步次数", m.stats.TotalSyncCount)),
		m.theme.Card.Render(fmt.Sprintf("📦\n %d\n 仓库数", m.stats.LastRepoCount)),
		m.theme.Card.Render(fmt.Sprintf("🆕\n %d\n 新增", m.stats.LastNewCount)),
	}
	cardRow := lipgloss.JoinHorizontal(lipgloss.Top, cards...)

	details := fmt.Sprintf("\n\n上次同步: %s\n耗时: %s", lastSync, duration)

	status := ""
	if m.syncing {
		status = m.theme.SidebarActive.Render("\n\n⏳ 同步中...")
	}
	help := m.theme.HelpText.Render("\n\ns: 开始同步  esc: 返回")

	return lipgloss.JoinVertical(lipgloss.Left,
		title, cardRow, details, status, help,
	)
}
