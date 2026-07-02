package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/types"
	"github.com/morehao/starman/internal/tui/styles"
)

type RepoDetailModel struct {
	store    store.Store
	theme    *styles.Theme
	repo     *store.Repository
	fullName string
	width    int
	height   int
	loaded   bool
}

type repoLoadedMsg struct{ repo *store.Repository }

func NewRepoDetail(s store.Store, theme *styles.Theme) *RepoDetailModel {
	return &RepoDetailModel{store: s, theme: theme}
}

func (m *RepoDetailModel) Init() tea.Cmd { return nil }

func (m *RepoDetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case types.RepoSelectedMsg:
		m.fullName = msg.FullName
		m.loaded = false
		return m, m.loadRepoCmd
	case repoLoadedMsg:
		m.repo = msg.repo
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return types.NavigatedMsg{Page: types.PageRepoList} }
		}
	}
	return m, nil
}

func (m *RepoDetailModel) loadRepoCmd() tea.Msg {
	ctx := context.Background()
	repo, err := m.store.GetRepository(ctx, m.fullName)
	if err != nil {
		return err
	}
	return repoLoadedMsg{repo: repo}
}

func (m *RepoDetailModel) View() string {
	if !m.loaded {
		return "loading repo..."
	}
	r := m.repo
	title := m.theme.PageTitle.Render(r.FullName) + "  \u2b50 " + fmt.Sprintf("%d", r.StargazersCount)
	meta := fmt.Sprintf("\nLanguage: %s | Forks: %d | Open Issues: %d",
		r.Language, r.ForksCount, 0)
	desc := fmt.Sprintf("\n\n%s", r.Description)
	aiSummary := ""
	if r.AISummary != "" {
		aiSummary = fmt.Sprintf("\n\nAI Summary:\n%s", r.AISummary)
	}
	tags := ""
	if len(r.CustomTags) > 0 {
		tags = fmt.Sprintf("\n\nTags: %s", strings.Join(r.CustomTags, ", "))
	}
	cat := ""
	if r.CustomCategory != "" {
		lock := "unlocked"
		if r.CategoryLocked {
			lock = "locked"
		}
		cat = fmt.Sprintf("\nCategory: %s (%s)", r.CustomCategory, lock)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		title, meta, desc, aiSummary, tags, cat,
		m.theme.HelpText.Render("\n\nEsc: back  s: star"),
	)
}
