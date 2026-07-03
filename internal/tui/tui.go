package tui

import (
	"context"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/app"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/discovery"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components"
	"github.com/morehao/starman/internal/tui/pages"
	"github.com/morehao/starman/internal/tui/styles"
)

type TuiModel struct {
	config       *config.Config
	store        store.Store
	theme        *styles.Theme
	width        int
	height       int
	currentPage  PageID
	sidebar      *components.SidebarModel
	statusbar    *components.StatusBarModel
	taskCenter   *TaskCenter
	syncAction   app.SyncAction
	searchAction app.SearchAction
	discovery    *discovery.Service
	pages        map[PageID]tea.Model
	ready        bool
	showHelp     bool
}

func NewTuiModel(cfg *config.Config, s store.Store) *TuiModel {
	theme := styles.DefaultTheme()

	ghToken := config.ResolveToken(cfg, "")
	gh := github.New(ghToken)
	ds := discovery.NewService(gh)

	syncAction := app.NewSyncAction(s, gh, cfg.GitHub.Username)

	aiKey := config.ResolveAIKey(cfg, "")
	aiClient := ai.NewClient(cfg.AI.BaseURL, aiKey, cfg.AI.Model)
	embeddingKey := config.ResolveEmbeddingKey(cfg, "")
	embeddingClient := ai.NewEmbeddingClient(cfg.Embedding.BaseURL, embeddingKey, cfg.Embedding.Model)
	aiSvc := ai.NewServiceWithEmbedding(aiClient, gh, embeddingClient)
	searchIndex := ai.NewSearchIndex()
	if s != nil {
		_ = searchIndex.Load(context.Background(), s)
	}
	aiSvc.SetSearchIndex(searchIndex)
	searchAction := app.NewSearchAction(s, aiSvc)

	m := &TuiModel{
		config:       cfg,
		store:        s,
		theme:        theme,
		currentPage:  PageDashboard,
		sidebar:      components.NewSidebar(theme),
		statusbar:    components.NewStatusBar(theme),
		taskCenter:   NewTaskCenter(),
		syncAction:   syncAction,
		searchAction: searchAction,
		discovery:    ds,
	}
	m.pages = map[PageID]tea.Model{
		PageDashboard:  pages.NewDashboard(s, theme),
		PageSearch:     pages.NewSearch(s, theme, searchAction),
		PageRepoList:   pages.NewRepoList(s, theme),
		PageRepoDetail: pages.NewRepoDetail(s, theme),
		PageTrending:   pages.NewTrending(s, theme, ds),
		PageSync:       pages.NewSync(s, theme, syncAction, m.taskCenter.Enqueue),
		PageAnalyze:    pages.NewAnalyze(s, theme),
		PageTag:        pages.NewTag(s, theme),
		PageCategorize: pages.NewCategorize(s, theme),
		PageStats:      pages.NewStats(s, theme),
		PageRelease:    pages.NewRelease(s, theme),
		PageGenerate:   pages.NewGenerate(s, theme),
		PageBackup:     pages.NewBackup(s, theme),
		PageConfig:     pages.NewConfig(cfg, theme),
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

	case TaskStartedMsg:
		m.taskCenter.MarkRunning(msg.ID)
		m.statusbar.SetTaskSummary(m.taskCenter.Summary())
		return m, nil

	case TaskProgressMsg:
		m.statusbar.SetTaskSummary(m.taskCenter.Summary())
		return m, nil

	case TaskDoneMsg:
		m.taskCenter.MarkDone(msg.ID, msg.Err)
		m.statusbar.SetTaskSummary(m.taskCenter.Summary())
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
	case "/":
		m.showHelp = false
		return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
		m.showHelp = false
		return m, nil
	default:
		if page, ok := components.ShortcutPage(msg.String()); ok {
			m.showHelp = false
			return m, func() tea.Msg { return NavigatedMsg{Page: page} }
		}
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
	if m.showHelp {
		help := m.renderHelp()
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, help)
	}
	sidebar := m.theme.Sidebar.Render(m.sidebar.View())
	content := m.renderContent()
	status := m.statusbar.View()

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}

func (m *TuiModel) renderHelp() string {
	lines := []string{
		"  q / ctrl+c  Quit                Esc      Back/Cancel",
		"  /           Search              ?        This help",
		"  Sidebar     Shortcuts:",
		"    1: Dashboard   r: Repo List   t: Trending",
		"    s: Sync        a: Analyze     g: Tag",
		"    c: Categorize  S: Stats       R: Release",
		"    G: Generate    b: Backup      C: Config",
		"  up/down/j/k  Navigate list      Enter    Select/Confirm",
	}
	return m.theme.Card.Render(
		m.theme.PageTitle.Render("Keyboard Shortcuts") + "\n\n" +
			lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
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
