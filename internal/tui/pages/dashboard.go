package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type DashboardModel struct {
	store     store.Store
	theme     *styles.Theme
	repoCount int
	langStats []langStat
	lastSync  string
	width     int
	height    int
	loaded    bool
}

type langStat struct {
	Lang  string
	Count int
	Pct   float64
}

func NewDashboard(s store.Store, theme *styles.Theme) *DashboardModel {
	return &DashboardModel{
		store: s,
		theme: theme,
	}
}

func (m *DashboardModel) Init() tea.Cmd {
	return m.loadCmd
}

func (m *DashboardModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, err := m.store.ListRepositories(ctx)
	if err != nil {
		return err
	}
	langMap := map[string]int{}
	for _, r := range repos {
		langMap[r.Language]++
	}
	total := len(repos)
	var stats []langStat
	for lang, count := range langMap {
		stats = append(stats, langStat{Lang: lang, Count: count, Pct: float64(count) / float64(total) * 100})
	}
	// sort descending by count (simple bubble sort for small n)
	for i := 0; i < len(stats); i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Count > stats[i].Count {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	if len(stats) > 5 {
		stats = stats[:5]
	}
	return dashboardLoadedMsg{
		repoCount: total,
		langStats: stats,
	}
}

type dashboardLoadedMsg struct {
	repoCount int
	langStats []langStat
}

func (m *DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 2
		m.height = msg.Height - 2
	case dashboardLoadedMsg:
		m.repoCount = msg.repoCount
		m.langStats = msg.langStats
		m.loaded = true
	}
	return m, nil
}

func (m *DashboardModel) View() string {
	if !m.loaded {
		return "loading dashboard..."
	}

	title := m.theme.PageTitle.Render("Dashboard") + "\n\n"

	// stat cards
	cards := []string{
		m.theme.Card.Render(fmt.Sprintf("⭐ %d\n 仓库总数", m.repoCount)),
		m.theme.Card.Render(fmt.Sprintf("📅 —\n 新增(周)")),
		m.theme.Card.Render(fmt.Sprintf("🤖 —\n 待分析")),
	}
	cardRow := lipgloss.JoinHorizontal(lipgloss.Top, cards...)

	// language distribution
	langSection := m.theme.CardTitle.Render("📊 语言分布 (Top 5)") + "\n"
	for _, s := range m.langStats {
		bar := strings.Repeat("█", int(s.Pct/2))
		langSection += fmt.Sprintf("  %-8s %-30s %4.0f%% %3d\n", s.Lang, bar, s.Pct, s.Count)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		cardRow,
		"",
		langSection,
	)
}
