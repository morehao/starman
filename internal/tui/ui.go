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
	"github.com/morehao/starman/internal/tui/components/footer"
	"github.com/morehao/starman/internal/tui/components/releasessection"
	"github.com/morehao/starman/internal/tui/components/repoview"
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
	showSidebar bool
	showHelp    bool
	ready       bool

	searching    bool
	searchQuery  string
	commandMode  bool
	errorMsg     string
	errorTimer   *time.Timer

	editingMode      string
	editingCats      []*store.Category
	editingCatCursor int
	editingQuery     string
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
		showSidebar: ctx.SidebarOpen,
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
	switch {
	case key.Matches(typed, m.keys.Escape):
		if m.editingMode != "" {
			m.editingMode = ""
			m.editingCats = nil
			m.editingCatCursor = 0
			m.editingQuery = ""
			return nil
		}
		if m.searching || m.commandMode {
			m.searching = false
			m.commandMode = false
			m.searchQuery = ""
			return nil
		}
		return nil

	case key.Matches(typed, m.keys.Quit):
		return tea.Quit

	case m.commandMode && key.Matches(typed, m.keys.Enter):
		cmd := parseCommand(m.searchQuery)
		m.commandMode = false
		m.searchQuery = ""
		return m.executeCommand(cmd)

	case m.commandMode:
		return m.handleSearchInput(typed)

	case m.searching && key.Matches(typed, m.keys.Enter):
		return m.executeSearch()

	case m.searching:
		return m.handleSearchInput(typed)

	case key.Matches(typed, m.keys.Command):
		m.commandMode = true
		m.searchQuery = ""
		return nil

	case key.Matches(typed, m.keys.Search):
		m.searching = true
		m.searchQuery = ""
		return nil

	case key.Matches(typed, m.keys.Sync):
		return m.startSync()

	case key.Matches(typed, m.keys.ToggleStar):
		if m.editingMode != "" || m.ctx.View != tuicontext.StarsView {
			return nil
		}
		return m.toggleStar()

	case key.Matches(typed, m.keys.EditCategory):
		if m.editingMode != "" || m.ctx.View != tuicontext.StarsView {
			return nil
		}
		return m.startEditCategory()

	case key.Matches(typed, m.keys.EditTag):
		if m.editingMode != "" || m.ctx.View != tuicontext.StarsView {
			return nil
		}
		return m.startEditTag()

	case m.editingMode == "category" && key.Matches(typed, m.keys.Enter):
		return m.finishEditCategory()

	case m.editingMode == "category" && key.Matches(typed, m.keys.Down):
		if m.editingCatCursor < len(m.editingCats)-1 {
			m.editingCatCursor++
		}
		return nil

	case m.editingMode == "category" && key.Matches(typed, m.keys.Up):
		if m.editingCatCursor > 0 {
			m.editingCatCursor--
		}
		return nil

	case m.editingMode == "tag" && key.Matches(typed, m.keys.Enter):
		return m.finishEditTag()

	case m.editingMode == "tag":
		return m.handleEditingInput(typed)

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

func (m *Model) handleSearchInput(typed tea.KeyMsg) tea.Cmd {
	k := typed.Key()
	switch k.String() {
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

func (m *Model) executeSearch() tea.Cmd {
	query := strings.TrimSpace(m.searchQuery)
	m.searching = false
	if query == "" || m.ctx.View != tuicontext.StarsView {
		m.searchQuery = ""
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

	mainHeight := h - constants.TabsHeight - constants.FooterHeight
	if mainHeight < 3 {
		mainHeight = 3
	}
	m.ctx.MainContentHeight = mainHeight

	if m.showSidebar && m.ctx.PreviewPosition == "right" {
		ratio := 0.38
		if m.ctx.TUICfg != nil && m.ctx.TUICfg.Preview.Width > 0 {
			ratio = m.ctx.TUICfg.Preview.Width
		}
		sidebarWidth := int(float64(w) * ratio)
		if sidebarWidth < 20 {
			sidebarWidth = 20
		}
		if sidebarWidth > w-30 {
			sidebarWidth = w - 30
		}
		m.ctx.DynamicPreviewWidth = sidebarWidth
		m.ctx.MainContentWidth = w - sidebarWidth
	} else {
		m.ctx.MainContentWidth = w
		m.ctx.DynamicPreviewWidth = 0
	}

	if m.showSidebar {
		m.sidebar.SetSize(m.ctx.DynamicPreviewWidth, mainHeight)
	}

	if ss, ok := m.currSection.(interface{ SetSize(int, int) }); ok {
		ss.SetSize(m.ctx.MainContentWidth, mainHeight)
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
	if m.showSidebar && m.ctx.DynamicPreviewWidth > 0 {
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
	} else {
		content = sectionView
	}

	tabsView := m.tabs.View()

	if m.editingMode == "category" {
		content = m.renderCategoryDialog()
	} else if m.editingMode == "tag" {
		content = m.renderTagDialog()
	}

	mainArea := lipgloss.JoinVertical(lipgloss.Left, tabsView, content)

	searchLine := ""
	if m.commandMode {
		searchLine = m.renderInputLine(":", m.searchQuery)
	} else if m.searching {
		searchLine = m.renderInputLine("Search: ", m.searchQuery)
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
	m.editingMode = "category"
	m.editingCats = cats
	m.editingCatCursor = 0
	row := m.currSection.CurrRow()
	if repoRow, ok := row.(starssection.RepoRow); ok && repoRow.Repo != nil {
		currentCat := repoRow.Repo.CustomCategory
		if currentCat == "" {
			currentCat = repoRow.Repo.AICategory
		}
		for i, c := range cats {
			if c.ID == currentCat {
				m.editingCatCursor = i
				break
			}
		}
	}
	return nil
}

func (m *Model) finishEditCategory() tea.Cmd {
	m.editingMode = ""
	if m.editingCatCursor < 0 || m.editingCatCursor >= len(m.editingCats) {
		m.editingCats = nil
		return nil
	}
	selected := m.editingCats[m.editingCatCursor]
	m.editingCats = nil

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
				Category:       selected.ID,
				CategoryLocked: repoRow.Repo.CategoryLocked,
			}); err != nil {
				return TaskFinishedMsg{TaskID: taskID, Name: "categorize", Message: "categorize failed", Err: err}
			}
			repoRow.Repo.CustomCategory = selected.ID
			return TaskFinishedMsg{
				TaskID:  taskID,
				Name:    "categorize",
				Message: fmt.Sprintf("category set to %s for %s", selected.ID, repoRow.Repo.FullName),
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
	m.editingMode = "tag"
	m.editingQuery = strings.Join(repoRow.Repo.CustomTags, ", ")
	return nil
}

func (m *Model) finishEditTag() tea.Cmd {
	m.editingMode = ""
	query := strings.TrimSpace(m.editingQuery)
	m.editingQuery = ""

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

func (m *Model) handleEditingInput(typed tea.KeyMsg) tea.Cmd {
	k := typed.Key()
	switch k.String() {
	case "backspace":
		if len(m.editingQuery) > 0 {
			m.editingQuery = m.editingQuery[:len(m.editingQuery)-1]
		}
	default:
		if k.Text != "" {
			m.editingQuery += k.Text
		} else if k.Code >= 32 && k.Code < 127 {
			m.editingQuery += string(k.Code)
		}
	}
	return nil
}

func (m Model) renderCategoryDialog() string {
	theme := m.ctx.Theme
	const dialogWidth = 44

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.PrimaryText).
		Bold(true)
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.FaintText)
	contentWidth := m.ctx.MainContentWidth

	var listLines []string
	for i, c := range m.editingCats {
		prefix := "  "
		lineStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText)
		if i == m.editingCatCursor {
			prefix = "▸ "
			lineStyle = lineStyle.Bold(true).Foreground(theme.SuccessText)
		}
		listLines = append(listLines, prefix+lineStyle.Render(c.Name+" ("+c.ID+")"))
	}

	body := titleStyle.Render("Select Category") + "\n\n" +
		strings.Join(listLines, "\n") + "\n\n" +
		helpStyle.Render("j/k navigate · enter confirm · esc cancel")

	dialog := lipgloss.NewStyle().
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.SuccessText).
		Render(body)

	return lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.ctx.MainContentHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}

func (m Model) renderTagDialog() string {
	theme := m.ctx.Theme
	const dialogWidth = 50
	contentWidth := m.ctx.MainContentWidth

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.PrimaryText).
		Bold(true)
	inputStyle := lipgloss.NewStyle().
		Foreground(theme.SuccessText)
	helpStyle := lipgloss.NewStyle().
		Foreground(theme.FaintText)

	display := m.editingQuery
	if display == "" {
		display = "(empty)"
	}

	body := titleStyle.Render("Edit Tags") + "\n\n" +
		"Tags: " + inputStyle.Render(display+"▎") + "\n\n" +
		helpStyle.Render("+tag to add · -tag to remove · enter confirm · esc cancel")

	dialog := lipgloss.NewStyle().
		Width(dialogWidth).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.SuccessText).
		Render(body)

	return lipgloss.NewStyle().
		Width(contentWidth).
		Height(m.ctx.MainContentHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}


