package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/discovery"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type TrendingModel struct {
	store     store.Store
	theme     *styles.Theme
	discovery *discovery.Service
	repos     []*discovery.TrendingRepo
	cursor    int
	source    string
	since     string
	width     int
	height    int
	loaded    bool
}

type trendingLoadedMsg struct{ repos []*discovery.TrendingRepo }

func NewTrending(s store.Store, theme *styles.Theme, discovery *discovery.Service) *TrendingModel {
	return &TrendingModel{store: s, theme: theme, discovery: discovery, source: "rss", since: "weekly"}
}

func (m *TrendingModel) Init() tea.Cmd { return m.loadCmd }

func (m *TrendingModel) loadCmd() tea.Msg {
	if m.discovery == nil {
		return trendingLoadedMsg{repos: nil}
	}
	ctx := context.Background()
	repos, err := m.discovery.Trending(ctx, discovery.TrendingOpts{Since: m.since, Source: m.source, Top: 20})
	if err != nil {
		return trendingLoadedMsg{repos: nil}
	}
	return trendingLoadedMsg{repos: repos}
}

func (m *TrendingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case trendingLoadedMsg:
		m.repos = msg.repos
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.repos)-1)
		case "tab":
			if m.source == "rss" {
				m.source = "search"
			} else {
				m.source = "rss"
			}
			return m, m.loadCmd
		case "enter":
			if m.cursor < len(m.repos) {
				return m, func() tea.Msg {
					return types.RepoSelectedMsg{FullName: m.repos[m.cursor].FullName}
				}
			}
		}
	}
	return m, nil
}

func (m *TrendingModel) View() string {
	if !m.loaded {
		return "loading trending..."
	}
	title := m.theme.PageTitle.Render("Trending") + "\n"
	body := title + fmt.Sprintf("Source: %s (Tab to switch)\n\n", m.source)
	for i, r := range m.repos {
		line := fmt.Sprintf("  %-40s \u2b50 %-8d %s", truncate(r.FullName, 38), r.Stars, r.Language)
		if i == m.cursor {
			body += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			body += line + "\n"
		}
	}
	return body
}
