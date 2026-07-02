package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type statsTab int

const (
	tabLanguage statsTab = iota
	tabCategory
	tabTag
)

var tabNames = []string{"Language", "Category", "Tag"}

type StatsModel struct {
	store   store.Store
	theme   *styles.Theme
	tab     statsTab
	results []langStat
	width   int
	height  int
	loaded  bool
	total   int
}

func NewStats(s store.Store, theme *styles.Theme) *StatsModel {
	return &StatsModel{store: s, theme: theme}
}

func (m *StatsModel) Init() tea.Cmd { return m.loadCmd }

func (m *StatsModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, _ := m.store.ListRepositories(ctx)
	return reposLoadedMsg{repos: repos}
}

func (m *StatsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case reposLoadedMsg:
		m.buildStats(msg.repos)
		m.loaded = true
	case tea.KeyMsg:
		if msg.String() == "tab" {
			m.tab = (m.tab + 1) % 3
			ctx := context.Background()
			repos, _ := m.store.ListRepositories(ctx)
			m.buildStats(repos)
		}
	}
	return m, nil
}

func (m *StatsModel) buildStats(repos []*store.Repository) {
	m.total = len(repos)
	counter := map[string]int{}
	for _, r := range repos {
		var key string
		switch m.tab {
		case tabLanguage:
			key = r.Language
		case tabCategory:
			key = r.CustomCategory
			if key == "" {
				key = r.AICategory
			}
		case tabTag:
			for _, t := range r.CustomTags {
				t = strings.TrimSpace(t)
				if t != "" {
					counter[t]++
				}
			}
			continue
		}
		if key == "" {
			key = "unknown"
		}
		counter[key]++
	}
	m.results = nil
	for k, v := range counter {
		m.results = append(m.results, langStat{Lang: k, Count: v, Pct: float64(v) / float64(m.total) * 100})
	}
	for i := 0; i < len(m.results); i++ {
		for j := i + 1; j < len(m.results); j++ {
			if m.results[j].Count > m.results[i].Count {
				m.results[i], m.results[j] = m.results[j], m.results[i]
			}
		}
	}
	if len(m.results) > 10 {
		m.results = m.results[:10]
	}
}

func (m *StatsModel) View() string {
	if !m.loaded {
		return "loading stats..."
	}
	title := m.theme.PageTitle.Render("Stats") + "\n"
	tabs := ""
	for i, name := range tabNames {
		if statsTab(i) == m.tab {
			tabs += m.theme.SidebarActive.Render("[" + name + "]") + " "
		} else {
			tabs += "[" + name + "] "
		}
	}
	body := tabs + fmt.Sprintf("\n\n  Total repos: %d\n\n", m.total)
	for _, s := range m.results {
		bar := strings.Repeat("\u2588", int(s.Pct/2))
		body += fmt.Sprintf("  %-15s %-30s %5.0f%% %4d\n", s.Lang, bar, s.Pct, s.Count)
	}
	return title + body
}
