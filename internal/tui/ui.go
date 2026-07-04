package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/common"
	"github.com/morehao/starman/internal/tui/components/actionsmenu"
	"github.com/morehao/starman/internal/tui/components/categoriessection"
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
	"github.com/morehao/starman/internal/tui/theme"
)

const taskClearDelay = 3 * time.Second

const (
	modeNormal = iota
	modeSearch
	modeCommand
	modePrompt
	modeActions
)

type Model struct {
	ctx         *tuicontext.ProgramContext
	tabs        tabs.Model
	sidebar     sidebar.Model
	footer      footer.Model
	stars       *starssection.Model
	categories  *categoriessection.Model
	trending    *trendingsection.Model
	releases    *releasessection.Model
	stats       *statssection.Model
	currSection section.Section
	actionsMenu actionsmenu.Model
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

	promptAction string
}

func NewModel(ctx *tuicontext.ProgramContext) Model {
	tabModel := tabs.NewModel(ctx)
	tabModel.SetTitles([]string{"Stars", "Categories", "Trending", "Releases", "Stats"})

	footerModel := footer.NewModel(ctx)

	starsModel := starssection.NewModel(1, ctx, section.SectionConfig{Title: "Stars"}, starssection.GroupAll)
	categoriesModel := categoriessection.NewModel(2, ctx, section.SectionConfig{Title: "Categories"})
	trendingModel := trendingsection.NewModel(3, ctx, section.SectionConfig{Title: "Trending"}, trendingsection.PeriodDaily)
	releasesModel := releasessection.NewModel(4, ctx, section.SectionConfig{Title: "Releases"}, releasessection.ShowUnread)
	statsModel := statssection.NewModel(5, ctx, section.SectionConfig{Title: "Stats"})

	tabModel.SetSectionTabs([]string{"\U0001F50D Search", "All", "Language", "Category", "Tag"})

	m := Model{
		ctx:         ctx,
		tabs:        tabModel,
		sidebar:     sidebar.NewModel(ctx),
		footer:      footerModel,
		stars:       starsModel,
		categories:  categoriesModel,
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
	m.drawer.SetTheme(ctx.Theme)
	m.searchInput.SetTheme(ctx.Theme)

	switch ctx.View {
	case tuicontext.StarsView:
		m.currSection = m.stars
	case tuicontext.CategoriesView:
		m.currSection = m.categories
	case tuicontext.TrendingView:
		m.currSection = m.trending
		m.tabs.SetSectionTabs([]string{"\U0001F50D Search", "Daily", "Weekly", "Monthly"})
	case tuicontext.ReleasesView:
		m.currSection = m.releases
		m.tabs.SetSectionTabs(nil)
	case tuicontext.StatsView:
		m.currSection = m.stats
		m.tabs.SetSectionTabs(nil)
		m.showSidebar = false
	}
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := m.stars.FetchNextPageSectionRows()
	cmds = append(cmds, m.categories.FetchNextPageSectionRows()...)
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
	updatedDrawer, drawerCmd := m.drawer.Update(msg)
	m.drawer = updatedDrawer.(drawer.Model)

	model, cmd := m.updateInner(msg)
	return model, maybeBatch(drawerCmd, cmd)
}

func (m Model) updateInner(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		} else {
			switch m.ctx.View {
			case tuicontext.StarsView:
				m.stars.ResetRows()
				m.currSection = m.stars
			case tuicontext.CategoriesView:
				m.categories.ResetRows()
				m.currSection = m.categories
			case tuicontext.TrendingView:
				m.trending.ResetRows()
				m.currSection = m.trending
			case tuicontext.ReleasesView:
				m.releases.ResetRows()
				m.currSection = m.releases
			}
			cmds := m.currSection.FetchNextPageSectionRows()
			return m, tea.Batch(append(cmds, clearAfterDelay(typed.TaskID))...)
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

	case categoriessection.CategoriesFetchedMsg:
		updated, cmd := m.categories.Update(typed)
		m.categories = updated.(*categoriessection.Model)
		m.syncSidebar()
		if typed.Err != nil {
			m.setError("categories fetch failed: " + typed.Err.Error())
		}
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
		if m.mode == modeActions {
			updated, cmd := m.actionsMenu.Update(typed)
			m.actionsMenu = updated.(actionsmenu.Model)
			if cmd != nil {
				msg := cmd()
				if result, ok := msg.(actionsmenu.ActionsMenuResultMsg); ok {
					m.mode = modeNormal
					if result.Confirmed {
						return m, m.handleActionsMenuResult(result)
					}
				}
			}
			if !m.actionsMenu.IsFocused() {
				m.mode = modeNormal
			}
			return m, nil
		}
		cmd := m.handleKey(typed)
		return m, cmd

	case tea.MouseClickMsg:
		return m, m.handleMouseClick(typed)
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
	case key.Matches(typed, m.ctx.Keys.Escape):
		return nil

	case key.Matches(typed, m.ctx.Keys.Quit):
		return tea.Quit

	case key.Matches(typed, m.ctx.Keys.Command):
		if m.mode == modeNormal {
			m.mode = modeCommand
			m.searchQuery = ""
		}
		return nil

	case key.Matches(typed, m.ctx.Keys.Search):
		if m.mode == modeNormal {
			m.mode = modeSearch
			m.searchInput.SetFocused(true)
		}
		return nil

	case key.Matches(typed, m.ctx.Keys.NextGroup):
		if m.ctx.View == tuicontext.StarsView {
			m.tabs.NextSection()
			m.stars.SetGroupBy(sectionIndexToGroupBy(m.tabs.ActiveSectionIndex()))
		}
		if m.ctx.View == tuicontext.TrendingView {
			m.tabs.NextSection()
			m.trending.SetPeriod(sectionIndexToTrendingPeriod(m.tabs.ActiveSectionIndex()))
			m.trending.ResetRows()
			return tea.Batch(m.trending.FetchNextPageSectionRows()...)
		}
		return nil

	case key.Matches(typed, m.ctx.Keys.PrevGroup):
		if m.ctx.View == tuicontext.StarsView {
			m.tabs.PrevSection()
			m.stars.SetGroupBy(sectionIndexToGroupBy(m.tabs.ActiveSectionIndex()))
		}
		if m.ctx.View == tuicontext.TrendingView {
			m.tabs.PrevSection()
			m.trending.SetPeriod(sectionIndexToTrendingPeriod(m.tabs.ActiveSectionIndex()))
			m.trending.ResetRows()
			return tea.Batch(m.trending.FetchNextPageSectionRows()...)
		}
		return nil

	case key.Matches(typed, m.ctx.Keys.NextView):
		m.switchView(1)
	case key.Matches(typed, m.ctx.Keys.PrevView):
		m.switchView(-1)
	case key.Matches(typed, m.ctx.Keys.Down):
		m.currSection.NextRow()
		m.syncSidebar()
	case key.Matches(typed, m.ctx.Keys.Up):
		m.currSection.PrevRow()
		m.syncSidebar()
	case key.Matches(typed, m.ctx.Keys.FirstLine):
		m.currSection.FirstItem()
		m.syncSidebar()
	case key.Matches(typed, m.ctx.Keys.LastLine):
		m.currSection.LastItem()
		m.syncSidebar()
	case key.Matches(typed, m.ctx.Keys.PrevSection):
		if m.ctx.View == tuicontext.StatsView {
			m.stats.PrevTab()
		} else {
			m.repo.PrevTab()
			m.syncSidebar()
		}
	case key.Matches(typed, m.ctx.Keys.NextSection):
		if m.ctx.View == tuicontext.StatsView {
			m.stats.NextTab()
		} else {
			m.repo.NextTab()
			m.syncSidebar()
		}
	case key.Matches(typed, m.ctx.Keys.ToggleSidebar):
		m.showSidebar = !m.showSidebar
		m.ctx.SidebarOpen = m.showSidebar
		m.recalcLayout()
	case key.Matches(typed, m.ctx.Keys.Help):
		m.showHelp = !m.showHelp

	case key.Matches(typed, m.ctx.Keys.ActionsMenu):
		return m.openActionsMenu()
	}
	return nil
}

func (m *Model) handleMouseClick(msg tea.MouseClickMsg) tea.Cmd {
	if msg.Button != tea.MouseLeft {
		return nil
	}

	y := msg.Y
	if y == 0 {
		if viewIdx := m.tabs.ViewTabAtX(msg.X); viewIdx >= 0 {
			m.switchView(viewIdx - m.tabs.Active())
		}
		return nil
	}

	if y == 1 && m.tabs.HasSectionTabs() {
		if secIdx := m.tabs.SectionTabAtX(msg.X); secIdx >= 0 {
			m.tabs.SetActiveSection(secIdx)
			if m.ctx.View == tuicontext.StarsView {
				m.stars.SetGroupBy(sectionIndexToGroupBy(secIdx))
			}
			if m.ctx.View == tuicontext.TrendingView {
				m.trending.SetPeriod(sectionIndexToTrendingPeriod(secIdx))
				m.trending.ResetRows()
				return tea.Batch(m.trending.FetchNextPageSectionRows()...)
			}
		}
		return nil
	}

	if y == m.ctx.ScreenHeight-1 {
		if viewIdx := m.footer.ViewSwitcherAtX(msg.X); viewIdx >= 0 {
			m.switchView(viewIdx - m.tabs.Active())
		}
		return nil
	}

	return nil
}



func (m *Model) repoNameFromRow() string {
	row := m.currSection.CurrRow()
	if repoRow, ok := row.(starssection.RepoRow); ok && repoRow.Repo != nil {
		return repoRow.Repo.FullName
	}
	return row.GetTitle()
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
		if m.ctx.View == tuicontext.CategoriesView {
			m.categories.ResetFilters()
		}
	}
	return nil
}

func (m *Model) handleCommandMode(typed tea.KeyMsg) tea.Cmd {
	// TODO: Extract command mode to components/commandmode/ as an independent tea.Model
	k := typed.Key()
	switch k.String() {
	case "esc":
		m.mode = modeNormal
		m.searchQuery = ""
		return nil
	case "enter":
		cmd := m.searchQuery
		m.mode = modeNormal
		m.searchQuery = ""
		if cmd == "" {
			return nil
		}
		if cmd == "q" || cmd == "quit" {
			return tea.Quit
		}
		if cmd == "help" {
			m.showHelp = !m.showHelp
			return nil
		}
		return m.executeCommand(cmd, cmd)
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
	switch m.promptAction {
	case "sync":
		if !result.Confirmed {
			return nil
		}
		cmdStr := "sync"
		if result.Confirmed {
			cmdStr = "sync --full"
		}
		return m.executeCommand(cmdStr, "syncing...")
	case "analyze":
		if result.Confirmed {
			return m.executeCommand("analyze --all", "analyzing all repos...")
		}
		return m.executeCommand("analyze", "analyzing repos...")
	case "categorize":
		if !result.Confirmed {
			return nil
		}
		repoName := m.repoNameFromRow()
		cmdStr := fmt.Sprintf("categorize %s %s", repoName, result.Value)
		return m.executeCommand(cmdStr, "categorizing...")
	case "tag":
		if !result.Confirmed {
			return nil
		}
		repoName := m.repoNameFromRow()
		cmdStr := fmt.Sprintf("tag %s %s", repoName, result.Value)
		return m.executeCommand(cmdStr, "tagging...")
	case "category_add":
		if !result.Confirmed {
			return nil
		}
		fields := strings.Split(result.Value, "\x00")
		if len(fields) < 5 || fields[0] == "" {
			m.setError("Category ID is required")
			return nil
		}
		cmdStr := fmt.Sprintf("category add %s --name %s --keywords %s --sort-order %s",
			fields[0], fields[1], fields[2], fields[3])
		if fields[4] == "true" {
			cmdStr += " --is-hidden"
		}
		return m.executeCommand(cmdStr, "adding category...")
	case "category_edit":
		if !result.Confirmed {
			return nil
		}
		fields := strings.Split(result.Value, "\x00")
		if len(fields) < 5 {
			return nil
		}
		cmdStr := fmt.Sprintf("category edit %s --name %s --keywords %s --sort-order %s",
			fields[0], fields[1], fields[2], fields[3])
		if fields[4] == "true" {
			cmdStr += " --is-hidden"
		}
		return m.executeCommand(cmdStr, "editing category...")
	case "category_delete":
		if !result.Confirmed {
			return nil
		}
		row := m.currSection.CurrRow()
		catRow, ok := row.(categoriessection.CategoryRow)
		if !ok || catRow.Category == nil {
			return nil
		}
		return m.executeCommand(
			fmt.Sprintf("category delete %s --force", catRow.Category.ID),
			"deleting category...",
		)
	}
	return nil
}

func (m *Model) executeCommand(cmdStr, statusText string) tea.Cmd {
	taskID := "cmd-" + time.Now().Format("150405")
	m.tasks.start(taskID, statusText)
	return func() tea.Msg {
		stdout, stderr, err := m.runner.Run(context.Background(), cmdStr)
		if errors.Is(err, ErrInteractiveRequired) {
			return TaskFinishedMsg{TaskID: taskID, Message: statusText, Err: fmt.Errorf("interactive commands must be run from CLI")}
		}
		if errors.Is(err, ErrBlockingRequired) {
			return TaskFinishedMsg{TaskID: taskID, Message: statusText, Err: fmt.Errorf("blocking commands must be run from CLI")}
		}
		m.drawer.AddEntry(":"+cmdStr, stdout, stderr)
		if err != nil {
			return TaskFinishedMsg{TaskID: taskID, Message: statusText + " failed", Err: fmt.Errorf("%s: %s", err.Error(), stderr)}
		}
		return TaskFinishedMsg{TaskID: taskID, Message: statusText + " done"}
	}
}

func sectionIndexToGroupBy(idx int) string {
	groups := []string{"all", "all", "language", "category", "tag"}
	if idx < 0 || idx >= len(groups) {
		return "all"
	}
	return groups[idx]
}

func sectionIndexToTrendingPeriod(idx int) string {
	switch idx {
	case 1:
		return trendingsection.PeriodDaily
	case 2:
		return trendingsection.PeriodWeekly
	case 3:
		return trendingsection.PeriodMonthly
	default:
		return trendingsection.PeriodDaily
	}
}

func (m *Model) executeSearch(query string) tea.Cmd {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if m.ctx.View == tuicontext.CategoriesView {
		m.categories.FilterRows(query)
		return nil
	}
	if m.ctx.View != tuicontext.StarsView {
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
		tuicontext.CategoriesView,
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
		m.tabs.SetSectionTabs([]string{"\U0001F50D Search", "All", "Language", "Category", "Tag"})
	case tuicontext.CategoriesView:
		m.currSection = m.categories
		m.tabs.SetSectionTabs([]string{"\U0001F50D Search", "All"})
	case tuicontext.TrendingView:
		m.currSection = m.trending
		m.tabs.SetSectionTabs([]string{"\U0001F50D Search", "Daily", "Weekly", "Monthly"})
	case tuicontext.ReleasesView:
		m.currSection = m.releases
		m.tabs.SetSectionTabs(nil)
	case tuicontext.StatsView:
		m.currSection = m.stats
		m.tabs.SetSectionTabs(nil)
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

	if m.mode == modeSearch {
		m.searchInput.SetSize(m.ctx.ScreenWidth, m.ctx.ScreenHeight)
		v := m.searchInput.View()
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	m.footer.SetPager(m.sectionPager())

	theme := m.ctx.Theme
	borderColor := theme.FaintBorder

	mainStyle := lipgloss.NewStyle().
		Width(m.ctx.MainContentWidth).
		Height(m.ctx.MainContentHeight)

	sectionView := mainStyle.Render(m.sectionView())

	var content string
	if m.mode == modeActions {
		content = m.actionsMenu.View().Content
	} else if m.mode == modePrompt {
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
	}

	helpLine := ""
	if m.showHelp {
		helpText := "j/k move  g/G first/last  h/l prev/next tab  "
		if m.tabs.HasSectionTabs() {
			helpText += "[ / ] prev/next section  "
		}
		helpText += "p sidebar  m actions  / search  : cmd  Tab view  ? help  q quit"
		helpLine = "\n" + common.RenderPreviewHeader(theme, m.ctx.ScreenWidth, helpText)
	}

	footerView := m.footer.View()

	extraLines := 1
	if searchLine != "" {
		extraLines++
	}
	if m.errorMsg != "" {
		extraLines++
	}
	if m.showHelp {
		extraLines++
	}

	fittingLines := m.ctx.ScreenHeight - extraLines
	if fittingLines < 0 {
		fittingLines = 0
	}
	adjustedMainArea, _ := truncateLines(mainArea, fittingLines)

	contentOutput := adjustedMainArea + searchLine + m.renderErrorBar() + helpLine

	v := tea.NewView(contentOutput + "\n" + footerView)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
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
		if m.ctx.View == tuicontext.CategoriesView {
			return m.renderCategoriesEmptyState()
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
	lines = append(lines, centerText(	fmt.Sprintf("   %s for actions menu, then choose Sync", boldStyle.Render("m")), w))
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

	m.repo.SetWidth(m.ctx.DynamicPreviewWidth)

	if repoRow, ok := row.(starssection.RepoRow); ok {
		m.repo.SetRepo(repoRow.Repo)
		m.sidebar.SetContent(m.repo.View())
	} else if catRow, ok := row.(categoriessection.CategoryRow); ok {
		m.sidebar.SetContent(renderCategoryDetail(catRow, m.ctx.Theme))
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

func maybeBatch(cmds ...tea.Cmd) tea.Cmd {
	var nonNil []tea.Cmd
	for _, c := range cmds {
		if c != nil {
			nonNil = append(nonNil, c)
		}
	}
	if len(nonNil) == 0 {
		return nil
	}
	if len(nonNil) == 1 {
		return nonNil[0]
	}
	return tea.Batch(nonNil...)
}

func truncateLines(s string, maxLines int) (string, bool) {
	if maxLines <= 0 {
		return "", false
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s, false
	}
	return strings.Join(lines[:maxLines], "\n"), true
}

func (m *Model) handleActionsMenuResult(result actionsmenu.ActionsMenuResultMsg) tea.Cmd {
	switch result.Action {
	case "category_add":
		m.promptAction = "category_add"
		fields := []prompt.FormField{
			prompt.NewFormField("ID", "", false, false),
			prompt.NewFormField("Name", "", false, false),
			prompt.NewFormField("Keywords", "", false, false),
			prompt.NewFormField("SortOrder", "0", false, false),
			prompt.NewFormField("IsHidden", "false", true, false),
		}
		m.prompt = prompt.NewCategoryFormModel("New Category", fields, false)
		m.prompt.SetTheme(m.ctx.Theme)
		m.mode = modePrompt

	case "category_edit":
		row := m.currSection.CurrRow()
		catRow, ok := row.(categoriessection.CategoryRow)
		if !ok || catRow.Category == nil {
			m.setError("No category selected")
			return nil
		}
		c := catRow.Category
		m.promptAction = "category_edit"
		hiddenStr := "false"
		if c.IsHidden {
			hiddenStr = "true"
		}
		kwStr := strings.Join(c.Keywords, ",")
		fields := []prompt.FormField{
			prompt.NewFormField("ID", c.ID, false, true),
			prompt.NewFormField("Name", c.Name, false, false),
			prompt.NewFormField("Keywords", kwStr, false, false),
			prompt.NewFormField("SortOrder", fmt.Sprintf("%d", c.SortOrder), false, false),
			prompt.NewFormField("IsHidden", hiddenStr, true, false),
		}
		m.prompt = prompt.NewCategoryFormModel("Edit Category", fields, true)
		m.prompt.SetTheme(m.ctx.Theme)
		m.mode = modePrompt

	case "category_delete":
		row := m.currSection.CurrRow()
		catRow, ok := row.(categoriessection.CategoryRow)
		if !ok || catRow.Category == nil {
			m.setError("No category selected")
			return nil
		}
		c := catRow.Category
		if !c.IsCustom {
			m.setError("Cannot delete built-in category: " + c.Name)
			return nil
		}
		m.promptAction = "category_delete"
		m.prompt = prompt.NewConfirmModel(fmt.Sprintf("Delete category '%s'?", c.Name))
		m.prompt.SetTheme(m.ctx.Theme)
		m.mode = modePrompt

	// ===== Stars =====
	case "toggle_star":
		return m.handleToggleStar()
	case "edit_category":
		return m.handleStarsEditCategory()
	case "edit_tag":
		return m.handleStarsEditTag()
	case "sync":
		return m.handleSyncAction()
	case "analyze":
		return m.handleAnalyzeAction()

	// ===== Trending =====
	case "refresh":
		m.currSection.ResetRows()
		return tea.Batch(m.currSection.FetchNextPageSectionRows()...)

	// ===== Releases =====
	case "mark_read":
		return m.releases.MarkCurrentRead()
	case "mark_all_read":
		return m.releases.MarkAllRead()
	case "toggle_filter":
		return m.handleReleaseFilterToggle()

	// ===== Common =====
	case "open_browser":
		return m.openInBrowser()
	}
	return nil
}

func (m *Model) openActionsMenu() tea.Cmd {
	hasRow := m.currSection.CurrRow() != nil

	var title string
	var items []actionsmenu.MenuItem

	switch m.ctx.View {
	case tuicontext.StarsView:
		title, items = m.buildStarsMenuItems(hasRow)
	case tuicontext.CategoriesView:
		title, items = m.buildCategoriesMenuItems(hasRow)
	case tuicontext.TrendingView:
		title, items = m.buildTrendingMenuItems(hasRow)
	case tuicontext.ReleasesView:
		title, items = m.buildReleasesMenuItems(hasRow)
	default:
		return nil
	}

	m.actionsMenu = actionsmenu.New(title, items, hasRow)
	m.actionsMenu.SetTheme(m.ctx.Theme)
	m.mode = modeActions
	return nil
}

func (m *Model) buildCategoriesMenuItems(hasRow bool) (string, []actionsmenu.MenuItem) {
	return "Category 操作", []actionsmenu.MenuItem{
		{Action: "category_add", Label: "+ 新增分类", NeedRow: false},
		{Action: "category_edit", Label: "✏ 编辑当前分类", NeedRow: true},
		{Action: "category_delete", Label: "✕ 删除当前分类", NeedRow: true},
	}
}

func (m *Model) buildStarsMenuItems(hasRow bool) (string, []actionsmenu.MenuItem) {
	starLabel := "Star"
	if hasRow {
		row := m.currSection.CurrRow()
		if repoRow, ok := row.(starssection.RepoRow); ok && repoRow.Repo != nil {
			if repoRow.Repo.StarredAt != "" {
				starLabel = "Unstar"
			}
		}
	}
	return "Stars 操作", []actionsmenu.MenuItem{
		{Action: "toggle_star", Label: starLabel, NeedRow: true},
		{Action: "edit_category", Label: "Edit Category", NeedRow: true},
		{Action: "edit_tag", Label: "Edit Tags", NeedRow: true},
		{Action: "---", Label: "────", NeedRow: false, Separator: true},
		{Action: "sync", Label: "Sync", NeedRow: false},
		{Action: "analyze", Label: "Analyze", NeedRow: false},
		{Action: "open_browser", Label: "Open in Browser", NeedRow: true},
	}
}

func (m *Model) buildTrendingMenuItems(hasRow bool) (string, []actionsmenu.MenuItem) {
	return "Trending 操作", []actionsmenu.MenuItem{
		{Action: "toggle_star", Label: "Star", NeedRow: true},
		{Action: "open_browser", Label: "Open in Browser", NeedRow: true},
		{Action: "---", Label: "────", NeedRow: false, Separator: true},
		{Action: "refresh", Label: "Refresh", NeedRow: false},
		{Action: "sync", Label: "Sync", NeedRow: false},
	}
}

func (m *Model) buildReleasesMenuItems(hasRow bool) (string, []actionsmenu.MenuItem) {
	filterLabel := "Show All"
	if m.releases.FilterLabel() == "All" {
		filterLabel = "Show Unread"
	}
	return "Releases 操作", []actionsmenu.MenuItem{
		{Action: "mark_read", Label: "Mark as Read", NeedRow: true},
		{Action: "mark_all_read", Label: "Mark All Read", NeedRow: false},
		{Action: "toggle_filter", Label: filterLabel, NeedRow: false},
		{Action: "---", Label: "────", NeedRow: false, Separator: true},
		{Action: "refresh", Label: "Refresh", NeedRow: false},
		{Action: "sync", Label: "Sync", NeedRow: false},
		{Action: "open_browser", Label: "Open in Browser", NeedRow: true},
	}
}

func (m *Model) handleToggleStar() tea.Cmd {
	repoName := m.repoNameFromRow()
	if repoName == "" {
		return nil
	}
	row := m.currSection.CurrRow()
	repoRow, ok := row.(starssection.RepoRow)
	if !ok || repoRow.Repo == nil {
		return m.executeCommand(fmt.Sprintf("star %s", repoName), "starring...")
	}
	if repoRow.Repo.StarredAt != "" {
		return m.executeCommand(fmt.Sprintf("unstar %s", repoName), "unstarring...")
	}
	return m.executeCommand(fmt.Sprintf("star %s", repoName), "starring...")
}

func (m *Model) handleStarsEditCategory() tea.Cmd {
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
	m.promptAction = "categorize"
	m.prompt = prompt.NewCategorySelectModel("Select category", names, currentCat)
	m.prompt.SetTheme(m.ctx.Theme)
	m.mode = modePrompt
	return nil
}

func (m *Model) handleStarsEditTag() tea.Cmd {
	row := m.currSection.CurrRow()
	repoRow, ok := row.(starssection.RepoRow)
	if !ok || repoRow.Repo == nil {
		return nil
	}
	currentTags := ""
	allTags := append([]string{}, repoRow.Repo.AITags...)
	allTags = append(allTags, repoRow.Repo.CustomTags...)
	currentTags = strings.Join(allTags, ",")
	m.promptAction = "tag"
	m.prompt = prompt.NewTagEditModel("Edit tags (+tag,-tag)", currentTags)
	m.prompt.SetTheme(m.ctx.Theme)
	m.mode = modePrompt
	return nil
}

func (m *Model) handleSyncAction() tea.Cmd {
	m.promptAction = "sync"
	m.prompt = prompt.NewConfirmModel("Run full sync? (y=--full, n=quick)")
	m.prompt.SetTheme(m.ctx.Theme)
	m.mode = modePrompt
	return nil
}

func (m *Model) handleAnalyzeAction() tea.Cmd {
	m.promptAction = "analyze"
	m.prompt = prompt.NewConfirmModel("Analyze all repos? (y=--all, n=incremental)")
	m.prompt.SetTheme(m.ctx.Theme)
	m.mode = modePrompt
	return nil
}

func (m *Model) handleReleaseFilterToggle() tea.Cmd {
	if m.releases.FilterLabel() == "All" {
		m.releases.SetFilter("unread")
	} else {
		m.releases.SetFilter("all")
	}
	m.releases.ResetRows()
	return tea.Batch(m.releases.FetchNextPageSectionRows()...)
}

func (m *Model) openInBrowser() tea.Cmd {
	row := m.currSection.CurrRow()
	if row == nil {
		return nil
	}
	url := row.GetUrl()
	if url == "" {
		m.setError("No URL available")
		return nil
	}

	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			return TaskFinishedMsg{Message: "open_browser", Err: fmt.Errorf("unsupported OS: %s", runtime.GOOS)}
		}
		if err := cmd.Start(); err != nil {
			return TaskFinishedMsg{Message: "open_browser", Err: fmt.Errorf("failed to open %s: %w", url, err)}
		}
		return TaskFinishedMsg{Message: "Opened " + url}
	}
}

func renderCategoryDetail(row categoriessection.CategoryRow, th theme.Theme) string {
	c := row.Category
	if c == nil {
		return ""
	}

	boldStyle := lipgloss.NewStyle().Foreground(th.PrimaryText).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(th.FaintText)

	var b strings.Builder
	b.WriteString(boldStyle.Render(c.Name))
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("ID: ") + c.ID + "\n")

	kwStr := strings.Join(c.Keywords, ", ")
	if kwStr == "" {
		kwStr = "-"
	}
	b.WriteString(dimStyle.Render("Keywords: ") + kwStr + "\n")
	b.WriteString(dimStyle.Render("Repos: ") + fmt.Sprintf("%d", row.RepoCount) + "\n")

	typeStr := "内置"
	if c.IsCustom {
		typeStr = "自定义"
	}
	b.WriteString(dimStyle.Render("Type: ") + typeStr + "\n")
	b.WriteString(dimStyle.Render("Sort order: ") + fmt.Sprintf("%d", c.SortOrder) + "\n")

	hiddenStr := "No"
	if c.IsHidden {
		hiddenStr = "Yes"
	}
	b.WriteString(dimStyle.Render("Hidden: ") + hiddenStr + "\n")

	return b.String()
}

func (m Model) renderCategoriesEmptyState() string {
	theme := m.ctx.Theme
	dimStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	boldStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)

	w := m.ctx.MainContentWidth
	if w < 40 {
		w = 40
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, centerText(boldStyle.Render("No categories yet"), w))
	lines = append(lines, "")
	lines = append(lines, centerText(dimStyle.Render("Press"), w))
	lines = append(lines, centerText(fmt.Sprintf("   %s to create a new category", boldStyle.Render("o")), w))
	lines = append(lines, "")

	return lipgloss.NewStyle().
		Width(w).Height(m.ctx.MainContentHeight).
		Render(strings.Join(lines, "\n"))
}


