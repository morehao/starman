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
	"github.com/morehao/starman/internal/generate"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/release"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components"
	"github.com/morehao/starman/internal/tui/pages"
	"github.com/morehao/starman/internal/tui/styles"
)

type TuiModel struct {
	config         *config.Config
	store          store.Store
	theme          *styles.Theme
	width          int
	height         int
	currentPage    PageID
	sidebar        *components.SidebarModel
	statusbar      *components.StatusBarModel
	taskCenter     *TaskCenter
	syncAction     app.SyncAction
	searchAction   app.SearchAction
	discovery      *discovery.Service
	batchAnalyzer  *ai.BatchAnalyzer
	generator      *generate.Generator
	releaseTracker *release.Tracker
	pages          map[PageID]tea.Model
	palette        *components.CommandPaletteModel
	workspace      *components.CommandWorkspaceModel
	uiState        UIState
	focusPane      FocusPane
	ready          bool
	showHelp       bool
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

	batchAnalyzer := ai.NewBatchAnalyzer(aiSvc, s, gh, 1)
	generator := generate.NewGenerator(s)
	releaseTracker := release.NewTracker(s, gh)

	m := &TuiModel{
		config:         cfg,
		store:          s,
		theme:          theme,
		currentPage:    PageDashboard,
		sidebar:        components.NewSidebar(theme),
		statusbar:      components.NewStatusBar(theme),
		taskCenter:     NewTaskCenter(),
		syncAction:     syncAction,
		searchAction:   searchAction,
		discovery:      ds,
		batchAnalyzer:  batchAnalyzer,
		generator:      generator,
		releaseTracker: releaseTracker,
		palette:        components.NewCommandPalette(theme, components.DefaultCommandCatalog()),
		workspace:      components.NewCommandWorkspace(theme),
		uiState:        StateNormal,
		focusPane:      FocusSidebar,
	}
	m.pages = map[PageID]tea.Model{
		PageDashboard:  pages.NewDashboard(s, theme),
		PageSearch:     pages.NewSearch(s, theme, searchAction),
		PageRepoList:   pages.NewRepoList(s, theme),
		PageRepoDetail: pages.NewRepoDetail(s, theme),
		PageTrending:   pages.NewTrending(s, theme, ds),
		PageSync:       pages.NewSync(s, theme, syncAction, m.taskCenter.Enqueue),
		PageAnalyze:    pages.NewAnalyze(s, theme, m.batchAnalyzer, m.taskCenter.Enqueue),
		PageTag:        pages.NewTag(s, theme),
		PageCategorize: pages.NewCategorize(s, theme),
		PageStats:      pages.NewStats(s, theme),
		PageRelease:    pages.NewRelease(s, theme, m.releaseTracker, m.taskCenter.Enqueue),
		PageGenerate:   pages.NewGenerate(s, theme, m.generator, cfg.GitHub.Username, m.taskCenter.Enqueue),
		PageBackup:     pages.NewBackup(s, theme, cfg, m.taskCenter.Enqueue),
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
		if m.uiState == StatePalette {
			return m.handlePaletteKey(msg)
		}
		return m.handleNormalKey(msg)
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

func (m *TuiModel) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+k":
		m.uiState = StateNormal
		m.palette.Close()
		return m, nil
	case "esc":
		m.uiState = StateNormal
		m.palette.Close()
		return m, nil
	case "enter":
		node, ok := m.palette.Selected()
		if !ok {
			return m, nil
		}
		return m.applySelectedCommand(node)
	default:
		_, _ = m.palette.Update(msg)
		return m, nil
	}
}

func (m *TuiModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "ctrl+k":
		m.uiState = StatePalette
		m.palette.Open()
		return m, nil
	case "/":
		m.showHelp = false
		return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "esc":
		m.showHelp = false
		return m, nil
	case "tab":
		if m.focusPane == FocusSidebar {
			m.focusPane = FocusWorkspace
		} else {
			m.focusPane = FocusSidebar
		}
		return m, nil
	default:
		if page, ok := components.ShortcutPage(msg.String()); ok {
			m.showHelp = false
			return m, func() tea.Msg { return NavigatedMsg{Page: page} }
		}
	}
	return m, nil
}

func (m *TuiModel) openPalette() {
	m.uiState = StatePalette
	m.palette.Open()
}

func (m *TuiModel) applySelectedCommand(node components.CommandNode) (tea.Model, tea.Cmd) {
	m.uiState = StateNormal
	m.palette.Close()
	m.workspace.SelectCommand(node)
	m.sidebar.SetRecent([]string{node.ID})
	return m, func() tea.Msg { return NavigatedMsg{Page: node.Page} }
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
	workspace := m.workspace.View()
	content := m.renderContent()
	body := lipgloss.JoinHorizontal(lipgloss.Top, workspace, content)

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, body)
	if m.width < 90 {
		main = lipgloss.JoinVertical(lipgloss.Left, sidebar, body)
	}
	status := m.statusbar.View()
	view := lipgloss.JoinVertical(lipgloss.Left, main, status)

	if m.uiState == StatePalette {
		paletteView := m.palette.View(m.width, m.height)
		if paletteView != "" {
			return paletteView
		}
	}

	return view
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
