package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/common"
	"github.com/morehao/starman/internal/tui/components/footer"
	"github.com/morehao/starman/internal/tui/components/repoview"
	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/components/sidebar"
	"github.com/morehao/starman/internal/tui/components/starssection"
	"github.com/morehao/starman/internal/tui/components/tabs"
	"github.com/morehao/starman/internal/tui/constants"
	tuicontext "github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/keys"
)

type Model struct {
	ctx         *tuicontext.ProgramContext
	keys        *keys.KeyMap
	tabs        tabs.Model
	sidebar     sidebar.Model
	footer      footer.Model
	stars       *starssection.Model
	repo        *repoview.Model
	showSidebar bool
	showHelp    bool
	ready       bool
}

func NewModel(ctx *tuicontext.ProgramContext) Model {
	keyMap := keys.Keys
	tabModel := tabs.NewModel(ctx)
	tabModel.SetTitles([]string{"Stars", "Trending", "Releases", "Stats"})

	footerModel := footer.NewModel(ctx)

	return Model{
		ctx:         ctx,
		keys:        &keyMap,
		tabs:        tabModel,
		sidebar:     sidebar.NewModel(ctx),
		footer:      footerModel,
		stars:       starssection.NewModel(1, ctx, section.SectionConfig{Title: "Stars"}, starssection.GroupAll),
		repo:        repoview.NewModel(),
		showSidebar: true,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := m.stars.FetchNextPageSectionRows()
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(typed)
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(typed, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(typed, m.keys.Down):
			m.stars.NextRow()
			m.syncSidebar()
		case key.Matches(typed, m.keys.Up):
			m.stars.PrevRow()
			m.syncSidebar()
		case key.Matches(typed, m.keys.FirstLine):
			m.stars.FirstItem()
			m.syncSidebar()
		case key.Matches(typed, m.keys.LastLine):
			m.stars.LastItem()
			m.syncSidebar()
		case key.Matches(typed, m.keys.ToggleSidebar):
			m.showSidebar = !m.showSidebar
			m.ctx.SidebarOpen = m.showSidebar
			m.recalcLayout()
		case key.Matches(typed, m.keys.Help):
			m.showHelp = !m.showHelp
		}
	}

	updated, cmd := m.stars.Update(msg)
	m.stars = updated.(*starssection.Model)
	m.syncSidebar()
	return m, cmd
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
		sidebarWidth := int(float64(w) * 0.38)
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
	m.stars.SetSize(m.ctx.MainContentWidth, mainHeight)
}

func (m Model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	m.footer.SetPager(m.stars.Pager())

	theme := m.ctx.Theme
	borderColor := theme.FaintBorder

	mainStyle := lipgloss.NewStyle().
		Width(m.ctx.MainContentWidth).
		Height(m.ctx.MainContentHeight)

	starsView := mainStyle.Render(m.starsView())

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
			starsView,
			sidebarContent,
		)
	} else {
		content = starsView
	}

	tabsView := m.tabs.View()

	helpLine := ""
	if m.showHelp {
		helpLine = "\n" + common.RenderPreviewHeader(theme, m.ctx.ScreenWidth,
			"j/k move  g/G first/last  p sidebar  ? help  q quit")
	}

	footerView := m.footer.View()

	v := tea.NewView(
		lipgloss.JoinVertical(
			lipgloss.Left,
			tabsView,
			content,
		) + helpLine + "\n" + footerView,
	)
	v.AltScreen = true
	return v
}

func (m Model) starsView() string {
	view := m.stars.View()
	if strings.TrimSpace(view) == "" {
		return m.renderEmptyState()
	}
	return view
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
	row := m.stars.CurrRow()
	if row == nil {
		m.repo.SetRepo(nil)
		m.sidebar.SetContent("")
		return
	}

	if repoRow, ok := row.(starssection.RepoRow); ok {
		m.repo.SetRepo(repoRow.Repo)
	} else {
		m.repo.SetRepo(&store.Repository{FullName: row.GetTitle()})
	}
	m.sidebar.SetContent(m.repo.View())
}
