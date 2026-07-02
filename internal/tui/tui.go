package tui

import (
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components"
	"github.com/morehao/starman/internal/tui/pages"
	"github.com/morehao/starman/internal/tui/styles"
)

type TuiModel struct {
	config      *config.Config
	store       store.Store
	theme       *styles.Theme
	width       int
	height      int
	currentPage PageID
	sidebar     *components.SidebarModel
	statusbar   *components.StatusBarModel
	pages       map[PageID]tea.Model
	ready       bool
}

func NewTuiModel(cfg *config.Config, s store.Store) *TuiModel {
	theme := styles.DefaultTheme()
	m := &TuiModel{
		config:      cfg,
		store:       s,
		theme:       theme,
		currentPage: PageDashboard,
		sidebar:     components.NewSidebar(theme),
		statusbar:   components.NewStatusBar(theme),
	}
	m.pages = map[PageID]tea.Model{
		PageDashboard:  pages.NewDashboard(s, theme),
		PageSearch:     pages.NewSearch(s, theme),
		PageRepoList:   pages.NewRepoList(s, theme),
		PageRepoDetail: pages.NewRepoDetail(s, theme),
		PageTrending:   pages.NewTrending(s, theme),
		PageSync:       pages.NewSync(s, theme),
		PageAnalyze:    pages.NewAnalyze(s, theme),
	}
	return m
}

func (m *TuiModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.sidebar.Init(),
		m.statusbar.Init(),
		tickCmd(),
	}
	for _, p := range m.pages {
		cmds = append(cmds, p.Init())
	}
	return tea.Batch(cmds...)
}

func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		for _, p := range m.pages {
			_, _ = p.Update(msg)
		}
		return m, nil

	case NavigatedMsg:
		m.currentPage = msg.Page
		return m, nil

	case RepoSelectedMsg:
		m.currentPage = PageRepoDetail
		return m, func() tea.Msg { return msg }

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "/":
			return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }
		}
	}

	_, cmd = m.sidebar.Update(msg)
	_, _ = m.statusbar.Update(msg)

	if p, ok := m.pages[m.currentPage]; ok {
		var pageCmd tea.Cmd
		_, pageCmd = p.Update(msg)
		cmd = tea.Batch(cmd, pageCmd)
	}
	return m, cmd
}

func (m *TuiModel) View() string {
	if !m.ready {
		return "loading..."
	}
	sidebar := m.theme.Sidebar.Render(m.sidebar.View())
	content := m.renderContent()
	status := m.statusbar.View()

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}

func (m *TuiModel) renderContent() string {
	p, ok := m.pages[m.currentPage]
	if !ok {
		return "page not found"
	}
	return p.View()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func RunTUI(cfg *config.Config) error {
	dir, err := config.DefaultDir()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "starman.db")
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	m := NewTuiModel(cfg, s)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
