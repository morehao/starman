package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/common"
	"github.com/morehao/starman/internal/tui/components/drawer"
	"github.com/morehao/starman/internal/tui/components/footer"
	"github.com/morehao/starman/internal/tui/components/prompt"
	"github.com/morehao/starman/internal/tui/components/releasessection"
	"github.com/morehao/starman/internal/tui/components/repoview"
	"github.com/morehao/starman/internal/tui/components/searchinput"
	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/components/sidebar"
	"github.com/morehao/starman/internal/tui/components/starssection"
	"github.com/morehao/starman/internal/tui/components/statssection"
	"github.com/morehao/starman/internal/tui/components/tabs"
	"github.com/morehao/starman/internal/tui/components/trendingsection"
	"github.com/morehao/starman/internal/tui/constants"
	tuicontext "github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/keys"
)

const taskClearDelay = 3 * time.Second

const (
	modeNormal = iota
	modeSearch
	modeCommand
	modePrompt
)

type Model struct {
	ctx         *tuicontext.ProgramContext
	keys        *keys.KeyMap
	tabs        tabs.Model
	sidebar     sidebar.Model
	footer      footer.Model
	stars       *starssection.Model
	trending    *trendingsection.Model
	releases    *releasessection.Model
	stats       *statssection.Model
	currSection section.Section
	repo        *repoview.Model
	tasks       *tasksHolder
	runner      CommandRunner
	drawer      drawer.Model
	showSidebar bool
	showHelp    bool
	ready       bool

	searchInput searchinput.Model
	prompt      prompt.Model
	mode        int
	searchQuery string

	errorMsg   string
	errorTimer *time.Timer
}

func NewModel(ctx *tuicontext.ProgramContext) Model {
	keyMap := keys.Keys
	tabModel := tabs.NewModel(ctx)
	tabModel.SetTitles([]string{"Stars", "Trending", "Releases", "Stats"})

	footerModel := footer.NewModel(ctx)

	starsModel := starssection.NewModel(1, ctx, section.SectionConfig{Title: "Stars"}, starssection.GroupAll)
	trendingModel := trendingsection.NewModel(2, ctx, section.SectionConfig{Title: "Trending"}, trendingsection.PeriodDaily)
	releasesModel := releasessection.NewModel(3, ctx, section.SectionConfig{Title: "Releases"}, releasessection.ShowUnread)
	statsModel := statssection.NewModel(4, ctx, section.SectionConfig{Title: "Stats"})

	m := Model{
		ctx:         ctx,
		keys:        &keyMap,
		tabs:        tabModel,
		sidebar:     sidebar.NewModel(ctx),
		footer:      footerModel,
		stars:       starsModel,
		trending:    trendingModel,
		releases:    releasesModel,
		stats:       statsModel,
		currSection: starsModel,
		repo:        repoview.NewModel(),
		tasks:       newTasksHolder(),
		drawer:      drawer.NewModel(),
		showSidebar: ctx.SidebarOpen,
		searchInput: searchinput.NewModel(),
		mode:        modeNormal,
	}

	switch ctx.View {
	case tuicontext.StarsView:
		m.currSection = m.stars
	case tuicontext.TrendingView:
		m.currSection = m.trending
	case tuicontext.ReleasesView:
		m.currSection = m.releases
	case tuicontext.StatsView:
		m.currSection = m.stats
		m.showSidebar = false
	}
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := m.stars.FetchNextPageSectionRows()
	cmds = append(cmds, m.trending.FetchNextPageSectionRows()...)
	cmds = append(cmds, m.releases.FetchNextPageSectionRows()...)
	cmds = append(cmds, m.stats.FetchNextPageSectionRows()...)
	cmds = append(cmds, tickSpinner())
	if len(cmds) == 0 {
		return tickSpinner()
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(typed)
		return m, nil

	case spinnerTickMsg:
		m.footer.SetTask(m.buildTaskInfo())
		if !m.tasks.isEmpty() {
			return m, tickSpinner()
		}
		return m, nil

	case TaskStartedMsg:
		m.tasks.start(typed.TaskID, typed.Name)
		return m, nil

	case TaskFinishedMsg:
		m.tasks.finish(typed.TaskID, typed.Message, typed.Err)
		m.footer.SetTask(m.buildTaskInfo())
		if typed.Err != nil {
			m.setError(typed.Message + ": " + typed.Err.Error())
		}
		if m.ctx.View == tuicontext.StarsView {
			m.stars.ResetRows()
			cmds := m.stars.FetchNextPageSectionRows()
			if len(cmds) > 0 {
				return m, tea.Batch(cmds...)
			}
		}
		return m, clearAfterDelay(typed.TaskID)

	case TaskClearedMsg:
		m.tasks.clear(typed.TaskID)
		m.footer.SetTask(m.buildTaskInfo())
		return m, nil

	case ErrorClearedMsg:
		m.errorMsg = ""
		return m, nil

	case starssection.ReposFetchedMsg:
		updated, cmd := m.stars.Update(typed)
		m.stars = updated.(*starssection.Model)
		m.currSection = m.stars
		m.syncSidebar()
		return m, cmd

	case starssection.ReposFetchFailedMsg:
		updated, cmd := m.stars.Update(typed)
		m.stars = updated.(*starssection.Model)
		m.currSection = m.stars
		m.syncSidebar()
		m.setError("fetch stars failed: " + typed.Err.Error())
		return m, cmd

	case trendingsection.TrendingFetchedMsg:
		updated, cmd := m.trending.Update(typed)
		m.trending = updated.(*trendingsection.Model)
		m.syncSidebar()
		if typed.Err != nil {
			m.setError("trending fetch failed: " + typed.Err.Error())
		}
		return m, cmd

	case releasessection.ReleasesFetchedMsg:
		updated, cmd := m.releases.Update(typed)
		m.releases = updated.(*releasessection.Model)
		m.syncSidebar()
		if typed.Err != nil {
			m.setError("releases fetch failed: " + typed.Err.Error())
		}
		return m, cmd

	case statssection.StatsFetchedMsg:
		updated, cmd := m.stats.Update(typed)
		m.stats = updated.(*statssection.Model)
		return m, cmd

	case tea.KeyMsg:
		cmd := m.handleKey(typed)
		return m, cmd
	}

	updated, cmd := m.currSection.Update(msg)
	m.currSection = updated
	m.syncSidebar()
	return m, cmd
}

func (m *Model) handleKey(typed tea.KeyMsg) tea.Cmd {
	switch m.mode {
	case modeSearch:
		return m.handleSearchMode(typed)
	case modeCommand:
		return m.handleCommandMode(typed)
	case modePrompt:
		return m.handlePromptMode(typed)
	}

	switch {
	case key.Matches(typed, m.keys.Escape):
		return nil

	case key.Matches(typed, m.keys.Quit):
		return tea.Quit

	case key.Matches(typed, m.keys.Command):
		if m.mode == modeNormal {
			m.mode = modeCommand
			m.searchQuery = ""
		}
		return nil

	case key.Matches(typed, m.keys.Search):
		if m.mode == modeNormal {
			m.mode = modeSearch
			m.searchInput.SetFocused(true)
		}
		return nil

	case key.Matches(typed, m.keys.Sync):
		return m.startSync()

	case key.Matches(typed, m.keys.ToggleStar):
		if m.ctx.View != tuicontext.StarsView {
			return nil
		}
		return m.toggleStar()

	case key.Matches(typed, m.keys.EditCategory):
		if m.ctx.View != tuicontext.StarsView {
			return nil
		}
		return m.startEditCategory()

	case key.Matches(typed, m.keys.EditTag):
		if m.ctx.View != tuicontext.StarsView {
			return nil
		}
		return m.startEditTag()

	case key.Matches(typed, m.keys.NextView):
		m.switchView(1)
	case key.Matches(typed, m.keys.PrevView):
		m.switchView(-1)
	case key.Matches(typed, m.keys.Down):
		m.currSection.NextRow()
		m.syncSidebar()
	case key.Matches(typed, m.keys.Up):
		m.currSection.PrevRow()
		m.syncSidebar()
	case key.Matches(typed, m.keys.FirstLine):
		m.currSection.FirstItem()
		m.syncSidebar()
	case key.Matches(typed, m.keys.LastLine):
		m.currSection.LastItem()
		m.syncSidebar()
	case key.Matches(typed, m.keys.PrevSection):
		if m.ctx.View == tuicontext.StatsView {
			m.stats.PrevTab()
		} else {
			m.repo.PrevTab()
			m.syncSidebar()
		}
	case key.Matches(typed, m.keys.NextSection):
		if m.ctx.View == tuicontext.StatsView {
			m.stats.NextTab()
		} else {
			m.repo.NextTab()
			m.syncSidebar()
		}
	case key.Matches(typed, m.keys.ToggleSidebar):
		m.showSidebar = !m.showSidebar
		m.ctx.SidebarOpen = m.showSidebar
		m.recalcLayout()
	case key.Matches(typed, m.keys.Help):
		m.showHelp = !m.showHelp
	}
	return nil
}

func (m *Model) handleSearchMode(typed tea.KeyMsg) tea.Cmd {
	updated, cmd := m.searchInput.Update(typed)
	m.searchInput = updated.(searchinput.Model)
	if cmd != nil {
		msg := cmd()
		if searchMsg, ok := msg.(searchinput.SearchExecutedMsg); ok {
			m.mode = modeNormal
			return m.executeSearch(searchMsg.Query)
		}
	}
	if !m.searchInput.IsFocused() {
		m.mode = modeNormal
	}
	return nil
}

func (m *Model) handleCommandMode(typed tea.KeyMsg) tea.Cmd {
	k := typed.Key()
	switch k.String() {
	case "esc":
		m.mode = modeNormal
		m.searchQuery = ""
		return nil
	case "enter":
		cmd := parseCommand(m.searchQuery)
		m.mode = modeNormal
		m.searchQuery = ""
		return m.executeCommand(cmd)
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		if k.Text != "" {
			m.searchQuery += k.Text
		} else if k.Code >= 32 && k.Code < 127 {
			m.searchQuery += string(k.Code)
		}
	}
	return nil
}

func (m *Model) handlePromptMode(typed tea.KeyMsg) tea.Cmd {
	updated, cmd := m.prompt.Update(typed)
	m.prompt = updated.(prompt.Model)
	if cmd != nil {
		msg := cmd()
		if result, ok := msg.(prompt.PromptResultMsg); ok {
			m.mode = modeNormal
			if !result.Confirmed {
				return nil
			}
			return m.handlePromptResult(result)
		}
	}
	if !m.prompt.IsFocused() {
		m.mode = modeNormal
	}
	return nil
}

func (m *Model) handlePromptResult(result prompt.PromptResultMsg) tea.Cmd {
	switch result.Type {
	case prompt.PromptCategorySelect:
		return m.finishEditCategoryWithValue(result.Value)
	case prompt.PromptTagEdit:
		return m.finishEditTagWithValue(result.Value)
	}
	return nil
}

func (m *Model) executeSearch(query string) tea.Cmd {
	query = strings.TrimSpace(query)
	if query == "" || m.ctx.View != tuicontext.StarsView {
		return nil
	}

	return func() tea.Msg {
		stdout, _, err := m.runner.Run(context.Background(), "search "+query+" --json")
		if err != nil {
			return starssection.ReposFetchedMsg{SectionID: 1, Repos: nil}
		}
		var hits []jsonHit
		if err := json.Unmarshal([]byte(stdout), &hits); err != nil {
			return starssection.ReposFetchedMsg{SectionID: 1, Repos: nil}
		}
		repos := convertSearchHitsToRepos(hits)
		return starssection.ReposFetchedMsg{SectionID: 1, Repos: repos}
	}
}

type jsonHit struct {
	Score    float64 `json:"score"`
	FullName string  `json:"full_name"`
	Language string  `json:"language"`
	Stars    int     `json:"stars"`
	Category string  `json:"category"`
	Summary  string  `json:"summary"`
}

func convertSearchHitsToRepos(hits []jsonHit) []*store.Repository {
	repos := make([]*store.Repository, len(hits))
	for i, h := range hits {
		repos[i] = &store.Repository{
			FullName:        h.FullName,
			Language:        h.Language,
			StargazersCount: h.Stars,
			AICategory:      h.Category,
			AISummary:       h.Summary,
		}
	}
	return repos
}

func (m *Model) startSync() tea.Cmd {
	cfg := m.ctx.Config
	if cfg == nil {
		return nil
	}
	token := config.ResolveToken(cfg, "")
	if token == "" || cfg.GitHub.Username == "" {
		errMsg := "sync requires github token and username"
		m.footer.SetTask(&footer.TaskInfo{
			Status:  2,
			Message: errMsg,
		})
		m.setError(errMsg)
		return clearAfterDelay("sync-err")
	}

	taskID := "sync-" + time.Now().Format("150405")
	m.tasks.start(taskID, "sync")

	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: "sync"} },
		func() tea.Msg {
			gh := github.New(token)
			repos, err := gh.ListStarred(context.Background(), cfg.GitHub.Username)
			if err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "sync", Message: "sync failed", Err: err}
			}
			if err := m.ctx.Store.UpsertReposOnSync(context.Background(), repos, false); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "sync", Message: "sync failed", Err: err}
			}
			return TaskFinishedMsg{
				TaskID:  taskID,
				Name:    "sync",
				Message: fmt.Sprintf("synced %d repos", len(repos)),
			}
		},
	)
}

func (m *Model) buildTaskInfo() *footer.TaskInfo {
	task := m.tasks.latest()
	if task == nil {
		return nil
	}
	frame := int(time.Now().UnixMilli()/120) % len(spinnerFrames)
	return &footer.TaskInfo{
		Status:     int(task.Status),
		Message:    task.Message,
		Err:        task.Err,
		SpinnerIdx: frame,
	}
}

func clearAfterDelay(taskID string) tea.Cmd {
	return tea.Tick(taskClearDelay, func(t time.Time) tea.Msg {
		return TaskClearedMsg{TaskID: taskID}
	})
}

func (m *Model) switchView(delta int) {
	views := []tuicontext.ViewType{
		tuicontext.StarsView,
		tuicontext.TrendingView,
		tuicontext.ReleasesView,
		tuicontext.StatsView,
	}
	currentIdx := 0
	for i, v := range views {
		if m.ctx.View == v {
			currentIdx = i
			break
		}
	}
	nextIdx := (currentIdx + delta + len(views)) % len(views)
	m.ctx.View = views[nextIdx]
	m.tabs.SetActive(nextIdx)

	switch m.ctx.View {
	case tuicontext.StarsView:
		m.currSection = m.stars
	case tuicontext.TrendingView:
		m.currSection = m.trending
	case tuicontext.ReleasesView:
		m.currSection = m.releases
	case tuicontext.StatsView:
		m.currSection = m.stats
		m.showSidebar = false
		m.ctx.SidebarOpen = false
		m.recalcLayout()
		return
	}
	m.showSidebar = true
	m.ctx.SidebarOpen = true
	m.recalcLayout()
	m.syncSidebar()
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.ctx.ScreenWidth = msg.Width
	m.ctx.ScreenHeight = msg.Height
	m.ready = true
	m.recalcLayout()
}

func (m *Model) recalcLayout() {
	w := m.ctx.ScreenWidth
	h := m.ctx.ScreenHeight
	if w <= 0 || h <= 0 {
		return
	}

	mainHeight := h - constants.TabsHeight - constants.FooterHeight
	m.ctx.MainContentHeight = mainHeight

	if m.drawer.IsOpen() {
		drawerHeight := int(float64(h) * 0.35)
		mainHeight -= drawerHeight
		m.ctx.MainContentHeight = mainHeight
	}

	if m.ctx.PreviewPosition == "auto" {
		if w < 50 {
			m.showSidebar = false
		} else if w < 80 {
			m.ctx.PreviewPosition = "bottom"
		} else {
			m.ctx.PreviewPosition = "right"
		}
	}

	if !m.showSidebar {
		m.ctx.MainContentWidth = w
		m.ctx.DynamicPreviewWidth = 0
		m.ctx.DynamicPreviewHeight = 0
		return
	}

	switch m.ctx.PreviewPosition {
	case "right":
		sidebarWidth := max(28, int(float64(w)*0.38))
		m.ctx.DynamicPreviewWidth = sidebarWidth
		m.ctx.DynamicPreviewHeight = mainHeight
		m.ctx.MainContentWidth = w - sidebarWidth
		m.sidebar.SetSize(m.ctx.DynamicPreviewWidth, mainHeight)
	case "bottom":
		sidebarHeight := int(float64(h) * 0.4)
		m.ctx.DynamicPreviewHeight = sidebarHeight
		m.ctx.DynamicPreviewWidth = w
		m.ctx.MainContentWidth = w
		m.ctx.MainContentHeight = mainHeight - sidebarHeight
		m.sidebar.SetSize(m.ctx.DynamicPreviewWidth, sidebarHeight)
	}

	if ss, ok := m.currSection.(interface{ SetSize(int, int) }); ok {
		ss.SetSize(m.ctx.MainContentWidth, m.ctx.MainContentHeight)
	}
}

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	m.footer.SetPager(m.sectionPager())

	theme := m.ctx.Theme
	borderColor := theme.FaintBorder

	mainStyle := lipgloss.NewStyle().
		Width(m.ctx.MainContentWidth).
		Height(m.ctx.MainContentHeight)

	sectionView := mainStyle.Render(m.sectionView())

	var content string
	if m.mode == modePrompt {
		content = m.prompt.View().Content
	} else if m.showSidebar && m.ctx.DynamicPreviewWidth > 0 && m.ctx.PreviewPosition == "right" {
		sidebarStyle := lipgloss.NewStyle().
			Width(m.ctx.DynamicPreviewWidth).
			Height(m.ctx.MainContentHeight).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(borderColor)

		sidebarContent := sidebarStyle.Render(m.sidebar.View())

		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			sectionView,
			sidebarContent,
		)
	} else if m.showSidebar && m.ctx.PreviewPosition == "bottom" {
		sidebarView := m.sidebar.View()
		content = lipgloss.JoinVertical(
			lipgloss.Top,
			sectionView,
			sidebarView,
		)
	} else {
		content = sectionView
	}

	tabsView := m.tabs.View()

	mainArea := lipgloss.JoinVertical(lipgloss.Left, tabsView, content)

	searchLine := ""
	switch m.mode {
	case modeCommand:
		searchLine = m.renderInputLine(":", m.searchQuery)
	case modeSearch:
		searchLine = "\n" + m.searchInput.View().Content
	}

	helpLine := ""
	if m.showHelp {
		helpLine = "\n" + common.RenderPreviewHeader(theme, m.ctx.ScreenWidth,
			"j/k move  g/G first/last  h/l prev/next tab  p sidebar  x star  c category  t tag  s sync  / search  : cmd  Tab view  ? help  q quit")
	}

	footerView := m.footer.View()

	v := tea.NewView(
		mainArea + searchLine + m.renderErrorBar() + helpLine + "\n" + footerView,
	)
	v.AltScreen = true
	return v
}

func (m Model) renderInputLine(prompt, query string) string {
	theme := m.ctx.Theme
	promptStyle := lipgloss.NewStyle().
		Foreground(theme.WarningText).
		Bold(true)
	inputStyle := lipgloss.NewStyle().
		Foreground(theme.PrimaryText)
	cursorStyle := lipgloss.NewStyle().
		Foreground(theme.SuccessText)

	return "\n" + promptStyle.Render(prompt) + inputStyle.Render(query) + cursorStyle.Render("▎")
}

func (m Model) sectionView() string {
	view := m.currSection.View()
	if strings.TrimSpace(view) == "" {
		if m.ctx.View == tuicontext.StarsView {
			return m.renderEmptyState()
		}
		return m.renderEmptyView(m.ctx.View)
	}
	return view
}

func (m Model) sectionPager() string {
	if p, ok := m.currSection.(interface{ Pager() string }); ok {
		return p.Pager()
	}
	return ""
}

func (m Model) renderEmptyView(view tuicontext.ViewType) string {
	theme := m.ctx.Theme
	dimStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	boldStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)

	w := m.ctx.MainContentWidth
	if w < 30 {
		w = 30
	}

	labels := map[tuicontext.ViewType]string{
		tuicontext.TrendingView: "Trending Repos",
		tuicontext.ReleasesView: "Release Updates",
		tuicontext.StatsView:    "Stats Dashboard",
	}

	label := labels[view]
	if label == "" {
		label = string(view)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, centerText(boldStyle.Render(label), w))
	lines = append(lines, "")
	lines = append(lines, centerText(dimStyle.Render("Coming soon..."), w))

	return lipgloss.NewStyle().
		Width(w).
		Height(m.ctx.MainContentHeight).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderEmptyState() string {
	theme := m.ctx.Theme
	dimStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	boldStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)
	starStyle := lipgloss.NewStyle().Foreground(theme.WarningText)

	w := m.ctx.MainContentWidth
	if w < 40 {
		w = 40
	}

	var lines []string
	lines = append(lines, "")
	welcome := fmt.Sprintf("%s  No repos yet", starStyle.Render("⭐"))
	lines = append(lines, centerText(welcome, w))
	lines = append(lines, "")
	lines = append(lines, centerText(dimStyle.Render("Your local database is empty. Press"), w))
	lines = append(lines, centerText(fmt.Sprintf("   %s to sync from GitHub", boldStyle.Render("s")), w))
	lines = append(lines, centerText(dimStyle.Render(":config init  to reconfigure"), w))
	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, "")

	return lipgloss.NewStyle().
		Width(w).
		Height(m.ctx.MainContentHeight).
		Render(strings.Join(lines, "\n"))
}

func centerText(text string, width int) string {
	textLen := lipgloss.Width(text)
	if textLen >= width {
		return text
	}
	pad := (width - textLen) / 2
	if pad < 0 {
		pad = 0
	}
	return strings.Repeat(" ", pad) + text
}

func (m *Model) syncSidebar() {
	if !m.showSidebar {
		return
	}
	row := m.currSection.CurrRow()
	if row == nil {
		m.repo.SetRepo(nil)
		m.sidebar.SetContent("")
		return
	}

	if repoRow, ok := row.(starssection.RepoRow); ok {
		m.repo.SetRepo(repoRow.Repo)
		m.sidebar.SetContent(m.repo.View())
	} else if tRow, ok := row.(trendingsection.TrendingRow); ok {
		m.repo.SetRepo(trendingsection.TrendingToStoreRepo(tRow.Repo))
		m.sidebar.SetContent(m.repo.View())
	} else if rRow, ok := row.(releasessection.ReleaseRow); ok {
		m.sidebar.SetContent(releasessection.ReleaseSummary(rRow.Release))
	} else {
		m.repo.SetRepo(&store.Repository{FullName: row.GetTitle()})
		m.sidebar.SetContent(m.repo.View())
	}
}

func (m *Model) setError(msg string) {
	m.errorMsg = msg
	if m.errorTimer != nil {
		m.errorTimer.Stop()
	}
	m.errorTimer = time.AfterFunc(5*time.Second, func() {
		m.errorMsg = ""
	})
}

func (m Model) renderErrorBar() string {
	if m.errorMsg == "" {
		return ""
	}
	theme := m.ctx.Theme
	return "\n" + lipgloss.NewStyle().
		Foreground(theme.ErrorText).
		Bold(true).
		Render("✖ "+m.errorMsg)
}

func (m *Model) toggleStar() tea.Cmd {
	row := m.currSection.CurrRow()
	repoRow, ok := row.(starssection.RepoRow)
	if !ok || repoRow.Repo == nil {
		return nil
	}
	repo := repoRow.Repo
	parts := strings.SplitN(repo.FullName, "/", 2)
	if len(parts) != 2 {
		m.setError("invalid repo full name: " + repo.FullName)
		return nil
	}
	cfg := m.ctx.Config
	token := config.ResolveToken(cfg, "")
	if token == "" {
		m.setError("GitHub token required for starring")
		return nil
	}

	taskID := "star-" + time.Now().Format("150405")
	isStarred := repo.StarredAt != ""
	action := "star"
	if isStarred {
		action = "unstar"
	}
	m.tasks.start(taskID, action+" "+repo.FullName)

	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: action + " " + repo.FullName} },
		func() tea.Msg {
			gh := github.New(token)
			ctx := context.Background()
			if isStarred {
				if err := gh.Unstar(ctx, parts[0], parts[1]); err != nil {
					return TaskFinishedMsg{TaskID: taskID, Name: action, Message: "unstar failed", Err: err}
				}
				repo.StarredAt = ""
			} else {
				if err := gh.Star(ctx, parts[0], parts[1]); err != nil {
					return TaskFinishedMsg{TaskID: taskID, Name: action, Message: "star failed", Err: err}
				}
				repo.StarredAt = time.Now().Format(time.RFC3339)
			}
			_ = m.ctx.Store.UpsertRepository(ctx, repo)
			return TaskFinishedMsg{
				TaskID:  taskID,
				Name:    action,
				Message: fmt.Sprintf("%s %s", action, repo.FullName),
			}
		},
	)
}

func (m *Model) startEditCategory() tea.Cmd {
	ctx := context.Background()
	cats, err := m.ctx.Store.ListCategories(ctx, true)
	if err != nil {
		m.setError("Failed to list categories: " + err.Error())
		return nil
	}
	names := make([]string, len(cats))
	for i, c := range cats {
		names[i] = c.Name
	}
	currentCat := ""
	row := m.currSection.CurrRow()
	if repoRow, ok := row.(starssection.RepoRow); ok && repoRow.Repo != nil {
		cat := repoRow.Repo.CustomCategory
		if cat == "" {
			cat = repoRow.Repo.AICategory
		}
		currentCat = cat
	}
	m.prompt = prompt.NewCategorySelectModel("Select category", names, currentCat)
	m.mode = modePrompt
	return nil
}

func (m *Model) finishEditCategoryWithValue(catName string) tea.Cmd {
	if catName == "" {
		return nil
	}
	cats, err := m.ctx.Store.ListCategories(context.Background(), true)
	if err != nil {
		m.setError("failed to list categories: " + err.Error())
		return nil
	}
	var catID string
	for _, c := range cats {
		if c.Name == catName {
			catID = c.ID
			break
		}
	}
	if catID == "" {
		return nil
	}

	row := m.currSection.CurrRow()
	repoRow, ok := row.(starssection.RepoRow)
	if !ok || repoRow.Repo == nil {
		return nil
	}

	taskID := "cat-" + time.Now().Format("150405")
	m.tasks.start(taskID, "categorize "+repoRow.Repo.FullName)

	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: "categorize " + repoRow.Repo.FullName} },
		func() tea.Msg {
			if err := m.ctx.Store.UpdateCustomFields(context.Background(), repoRow.Repo.ID, &store.CustomFields{
				Description:    repoRow.Repo.CustomDescription,
				Tags:           repoRow.Repo.CustomTags,
				Category:       catID,
				CategoryLocked: repoRow.Repo.CategoryLocked,
			}); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "categorize", Message: "categorize failed", Err: err}
			}
			repoRow.Repo.CustomCategory = catID
			return TaskFinishedMsg{
				TaskID:  taskID,
				Name:    "categorize",
				Message: fmt.Sprintf("category set to %s for %s", catID, repoRow.Repo.FullName),
			}
		},
	)
}

func (m *Model) startEditTag() tea.Cmd {
	row := m.currSection.CurrRow()
	repoRow, ok := row.(starssection.RepoRow)
	if !ok || repoRow.Repo == nil {
		return nil
	}
	currentTags := ""
	allTags := append([]string{}, repoRow.Repo.AITags...)
	allTags = append(allTags, repoRow.Repo.CustomTags...)
	currentTags = strings.Join(allTags, ",")
	m.prompt = prompt.NewTagEditModel("Edit tags (+tag,-tag)", currentTags)
	m.mode = modePrompt
	return nil
}

func (m *Model) finishEditTagWithValue(query string) tea.Cmd {
	query = strings.TrimSpace(query)

	row := m.currSection.CurrRow()
	repoRow, ok := row.(starssection.RepoRow)
	if !ok || repoRow.Repo == nil {
		return nil
	}

	var addTags, removeTags []string
	if query != "" {
		addTags, removeTags = store.ParseTagExpr(query)
	}
	newTags := store.ApplyTags(repoRow.Repo.CustomTags, addTags, removeTags)

	taskID := "tag-" + time.Now().Format("150405")
	m.tasks.start(taskID, "tag "+repoRow.Repo.FullName)

	return tea.Batch(
		func() tea.Msg { return TaskStartedMsg{TaskID: taskID, Name: "tag " + repoRow.Repo.FullName} },
		func() tea.Msg {
			if err := m.ctx.Store.UpdateCustomFields(context.Background(), repoRow.Repo.ID, &store.CustomFields{
				Description:    repoRow.Repo.CustomDescription,
				Tags:           newTags,
				Category:       repoRow.Repo.CustomCategory,
				CategoryLocked: repoRow.Repo.CategoryLocked,
			}); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "tag", Message: "tag update failed", Err: err}
			}
			repoRow.Repo.CustomTags = newTags
			return TaskFinishedMsg{
				TaskID:  taskID,
				Name:    "tag",
				Message: fmt.Sprintf("tags updated for %s", repoRow.Repo.FullName),
			}
		},
	)
}


