package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/types"
	"github.com/morehao/starman/internal/tui/styles"
)

type RepoListModel struct {
	store  store.Store
	theme  *styles.Theme
	repos  []*store.Repository
	cursor int
	width  int
	height int
	loaded bool
}

type reposLoadedMsg struct{ repos []*store.Repository }

func NewRepoList(s store.Store, theme *styles.Theme) *RepoListModel {
	return &RepoListModel{store: s, theme: theme}
}

func (m *RepoListModel) Init() tea.Cmd { return m.loadCmd }

func (m *RepoListModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, err := m.store.ListRepositories(ctx)
	if err != nil {
		return err
	}
	sortByStars(repos)
	return reposLoadedMsg{repos: repos}
}

func sortByStars(repos []*store.Repository) {
	for i := 0; i < len(repos); i++ {
		for j := i + 1; j < len(repos); j++ {
			if repos[j].StargazersCount > repos[i].StargazersCount {
				repos[i], repos[j] = repos[j], repos[i]
			}
		}
	}
}

func (m *RepoListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case reposLoadedMsg:
		m.repos = msg.repos
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.repos)-1)
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

func (m *RepoListModel) View() string {
	if !m.loaded {
		return "loading repos..."
	}
	title := m.theme.PageTitle.Render("Repo List") + "\n"
	header := fmt.Sprintf("  %-40s %-12s %8s %15s", "full_name", "language", "stars", "category")
	rendered := title + m.theme.TableHeader.Render(header) + "\n"
	start := max(m.cursor-m.height+5, 0)
	end := min(start+m.height-5, len(m.repos))
	for i := start; i < end; i++ {
		r := m.repos[i]
		line := fmt.Sprintf("  %-40s %-12s %8d %15s",
			truncate(r.FullName, 38), r.Language, r.StargazersCount, r.CustomCategory)
		if i == m.cursor {
			rendered += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			rendered += line + "\n"
		}
	}
	return rendered
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "\u2026"
	}
	return s
}
