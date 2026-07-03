package tui

import (
	"context"
	"path/filepath"
	"strings"
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
	"github.com/morehao/starman/internal/tui/types"
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
	recentIDs      []string
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
		_, _ = m.workspace.Update(msg)
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
		if m.workspace.SelectedID() != "" {
			m.sidebar.SetRunning(m.workspace.SelectedID(), true)
		}
		return m, nil

	case TaskProgressMsg:
		m.statusbar.SetTaskSummary(m.taskCenter.Summary())
		_, _ = m.workspace.Update(msg)
		return m, nil

	case TaskDoneMsg:
		m.taskCenter.MarkDone(msg.ID, msg.Err)
		m.statusbar.SetTaskSummary(m.taskCenter.Summary())
		if m.workspace.SelectedID() != "" {
			m.sidebar.SetRunning(m.workspace.SelectedID(), false)
		}
		return m, nil

	case tea.KeyMsg:
		if m.uiState == StatePalette {
			return m.handlePaletteKey(msg)
		}

		if m.currentPage == PageSearch {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "ctrl+k":
				m.openPalette()
				return m, nil
			case "/":
				m.showHelp = false
				return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }
			case "?":
				m.showHelp = !m.showHelp
				return m, nil
			}
			if p, ok := m.pages[PageSearch]; ok {
				_, pageCmd := p.Update(msg)
				_, _ = m.statusbar.Update(msg)
				return m, pageCmd
			}
		}

		model, handledCmd, consumed := m.handleNormalKey(msg)
		if consumed {
			return model, handledCmd
		}
		if m.focusPane == FocusWorkspace {
			_, wsCmd := m.workspace.Update(msg)
			return m, tea.Batch(handledCmd, wsCmd)
		}
		cmd = handledCmd
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

func (m *TuiModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit, true
	case "ctrl+k":
		m.openPalette()
		return m, nil, true
	case "/":
		m.showHelp = false
		return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }, true
	case "?":
		m.showHelp = !m.showHelp
		return m, nil, true
	case "esc":
		m.showHelp = false
		return m, nil, true
	case "ctrl+enter":
		if err := m.workspace.Validate(); err != nil {
			return m, func() tea.Msg {
				return types.StatusMsg{Text: err.Error(), Level: types.LevelError, Timeout: 4 * time.Second}
			}, true
		}
		id := m.workspace.SelectedID()
		taskID := m.taskCenter.Enqueue("execute " + id)
		m.workspace.AppendOutput("executing " + id + "...")
		m.statusbar.SetTaskSummary(m.taskCenter.Summary())
		m.sidebar.SetRunning(id, true)
		return m, tea.Batch(
			func() tea.Msg { return TaskStartedMsg{ID: taskID, Label: "execute " + id} },
			func() tea.Msg { return TaskDoneMsg{ID: taskID, Err: nil} },
			func() tea.Msg {
				return types.StatusMsg{Text: "executed successfully", Level: types.LevelSuccess, Timeout: 2 * time.Second}
			},
		), true
	case "tab":
		if m.focusPane == FocusSidebar {
			m.focusPane = FocusWorkspace
		} else {
			m.focusPane = FocusSidebar
		}
		return m, nil, true
	default:
		if m.currentPage != PageSearch {
			if page, ok := components.ShortcutPage(msg.String()); ok {
				m.showHelp = false
				return m, func() tea.Msg { return NavigatedMsg{Page: page} }, true
			}
		}
	}
	return m, nil, false
}

func (m *TuiModel) openPalette() {
	m.uiState = StatePalette
	m.palette.Open()
}

func (m *TuiModel) applySelectedCommand(node components.CommandNode) (tea.Model, tea.Cmd) {
	m.uiState = StateNormal
	m.palette.Close()
	m.workspace.SelectCommand(node)
	m.recentIDs = prependDedup(m.recentIDs, node.ID, 5)
	m.sidebar.SetRecent(m.recentIDs)
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

	w := m.width
	contentHeight := m.height - 5

	sbw := w * 3 / 10
	if sbw < 24 {
		sbw = 24
	}

	m.sidebar.SetRenderWidth(sbw)
	sidebar := m.theme.Sidebar.Width(sbw).Height(contentHeight).Render(m.sidebar.View())

	rightPane := m.renderContent()
	if m.workspace.SelectedID() != "" {
		rightPane = m.workspace.View()
	}
	contentStyle := lipgloss.NewStyle().
		Width(w - sbw - 3).
		Height(contentHeight).
		Padding(0, 1)
	rightPane = contentStyle.Render(rightPane)

	sbLines := splitLines(sidebar)
	ctLines := splitLines(rightPane)
	padLines(&sbLines, &ctLines, contentHeight)

	var lines []string
	lines = append(lines, renderTopBorder(w))
	lines = append(lines, m.renderHeaderLine(w))
	lines = append(lines, renderSep(w, sbw))

	for i := range sbLines {
		lines = append(lines, "│"+sbLines[i]+"│"+ctLines[i]+"│")
	}
	lines = append(lines, renderSep(w, -1))

	statusContent := m.statusbar.View()
	statusW := lipgloss.Width(statusContent)
	statusPad := w - 4 - statusW
	if statusPad < 0 {
		statusPad = 0
	}
	lines = append(lines, "│ "+statusContent+strings.Repeat(" ", statusPad)+" │")
	lines = append(lines, renderBottomBorder(w, sbw))

	view := strings.Join(lines, "\n")

	if m.uiState == StatePalette {
		paletteView := m.palette.View(m.width, m.height)
		if paletteView != "" {
			return overlay(view, paletteView)
		}
	}

	return view
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func padLines(a, b *[]string, target int) {
	for len(*a) < target {
		*a = append(*a, "")
	}
	for len(*b) < target {
		*b = append(*b, "")
	}
}

func renderTopBorder(w int) string {
	title := "STARMAN TUI"
	return "┌─ " + title + " ─" + strings.Repeat("─", w-len(title)-6) + "┐"
}

func renderSep(w int, sbw int) string {
	if sbw < 0 {
		return "├" + strings.Repeat("─", w-2) + "┤"
	}
	left := strings.Repeat("─", sbw)
	right := strings.Repeat("─", w-sbw-3)
	return "├" + left + "┬" + right + "┤"
}

func renderBottomBorder(w int, sbw int) string {
	if sbw < 0 {
		return "└" + strings.Repeat("─", w-2) + "┘"
	}
	left := strings.Repeat("─", sbw)
	right := strings.Repeat("─", w-sbw-3)
	return "└" + left + "┴" + right + "┘"
}

func (m *TuiModel) renderHelp() string {
	lines := []string{
		"  Ctrl+K       Palette            Tab       Switch focus",
		"  Ctrl+Enter   Execute command    Esc       Back/Cancel",
		"  q / ctrl+c   Quit               ?         This help",
		"",
		"  Sidebar     Shortcuts:",
		"    1: Dashboard   r: Repo List   t: Trending",
		"    s: Sync        a: Analyze     g: Tag",
		"    c: Categorize  S: Stats       R: Release",
		"    G: Generate    b: Backup      C: Config",
		"",
		"  up/down/j/k  Navigate list      Enter    Select/Confirm",
	}
	return m.theme.Card.Render(
		m.theme.PageTitle.Render("Keyboard Shortcuts") + "\n\n" +
			lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m *TuiModel) renderHeaderLine(w int) string {
	var leftContent string
	if node, ok := components.CommandByPage(m.currentPage); ok {
		leftContent = m.theme.CardTitle.Render(node.Group) +
			lipgloss.NewStyle().Foreground(m.theme.Subtle).Render(" ▸ ") +
			lipgloss.NewStyle().Foreground(m.theme.Text).Render(node.Label)
	} else {
		leftContent = lipgloss.NewStyle().Foreground(m.theme.Primary).Bold(true).Render("STARMAN")
	}

	rightContent := ""
	if summary := m.taskCenter.Summary(); summary != "" {
		rightContent += lipgloss.NewStyle().Foreground(m.theme.Warning).Render("[" + summary + "] ")
	}
	rightContent += lipgloss.NewStyle().Foreground(m.theme.Subtle).Render(time.Now().Format("15:04"))

	leftW := lipgloss.Width(leftContent)
	rightW := lipgloss.Width(rightContent)
	filler := w - 4 - leftW - rightW
	if filler < 0 {
		filler = 0
	}

	return "│ " + leftContent + strings.Repeat(" ", filler) + rightContent + " │"
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

func overlay(base, top string) string {
	baseLines := strings.Split(base, "\n")
	topLines := strings.Split(top, "\n")
	maxLines := len(baseLines)
	if len(topLines) > maxLines {
		maxLines = len(topLines)
	}
	var result []string
	for i := 0; i < maxLines; i++ {
		var baseLine, topLine string
		if i < len(baseLines) {
			baseLine = baseLines[i]
		}
		if i < len(topLines) {
			topLine = topLines[i]
		}
		baseRunes := []rune(baseLine)
		topRunes := []rune(topLine)
		for len(baseRunes) < len(topRunes) {
			baseRunes = append(baseRunes, ' ')
		}
		for j := 0; j < len(topRunes) && j < len(baseRunes); j++ {
			if topRunes[j] != ' ' {
				baseRunes[j] = topRunes[j]
			}
		}
		result = append(result, string(baseRunes))
	}
	return strings.Join(result, "\n")
}

func prependDedup(slice []string, val string, maxLen int) []string {
	result := []string{val}
	for _, s := range slice {
		if s == val {
			continue
		}
		result = append(result, s)
	}
	if len(result) > maxLen {
		result = result[:maxLen]
	}
	return result
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
