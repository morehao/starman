package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type AnalyzeModel struct {
	store     store.Store
	theme     *styles.Theme
	repos     []*store.Repository
	cursor    int
	width     int
	height    int
	loaded    bool
	analyzing bool
}

type unanalyzedMsg struct{ repos []*store.Repository }
type analyzeDoneMsg struct{ success, failed, total int }

func NewAnalyze(s store.Store, theme *styles.Theme) *AnalyzeModel {
	return &AnalyzeModel{store: s, theme: theme}
}

func (m *AnalyzeModel) Init() tea.Cmd { return m.loadCmd }

func (m *AnalyzeModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, err := m.store.ListUnanalyzed(ctx, 50)
	if err != nil {
		return unanalyzedMsg{repos: nil}
	}
	return unanalyzedMsg{repos: repos}
}

func (m *AnalyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case unanalyzedMsg:
		m.repos = msg.repos
		m.loaded = true
		m.analyzing = false
	case analyzeDoneMsg:
		m.analyzing = false
		return m, m.loadCmd
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.repos)-1)
		case "a":
			if !m.analyzing {
				m.analyzing = true
				return m, func() tea.Msg {
					return analyzeDoneMsg{success: 0, failed: 0, total: len(m.repos)}
				}
			}
		case "enter":
			if m.cursor < len(m.repos) {
				return m, func() tea.Msg {
					return types.RepoSelectedMsg{FullName: m.repos[m.cursor].FullName}
				}
			}
		case "esc":
			return m, func() tea.Msg { return types.NavigatedMsg{Page: types.PageDashboard} }
		}
	}
	return m, nil
}

func (m *AnalyzeModel) View() string {
	if !m.loaded {
		return "loading analyze..."
	}
	title := m.theme.PageTitle.Render("Analyze") + "\n\n"

	if m.analyzing {
		return title + m.theme.SidebarActive.Render("⏳ 分析中...")
	}

	body := fmt.Sprintf("待分析: %d\n\n", len(m.repos))
	for i, r := range m.repos {
		desc := truncate(r.Description, 50)
		line := fmt.Sprintf("  %-40s %-10s %s", truncate(r.FullName, 38), r.Language, desc)
		if i == m.cursor {
			body += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			body += line + "\n"
		}
	}

	help := m.theme.HelpText.Render("\na: 开始分析  enter: 查看  esc: 返回")
	return title + body + help
}
