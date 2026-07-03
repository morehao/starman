package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/footer"
	"github.com/morehao/starman/internal/tui/components/repoview"
	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/components/sidebar"
	"github.com/morehao/starman/internal/tui/components/starssection"
	"github.com/morehao/starman/internal/tui/components/tabs"
	tuicontext "github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/keys"
)

type Model struct {
	ctx         *tuicontext.ProgramContext
	keys        *keys.KeyMap
	tabs        tabs.Model
	sidebar     sidebar.Model
	footer      footer.Model
	stars       section.Section
	repo        *repoview.Model
	showSidebar bool
	showHelp    bool
}

func NewModel(ctx *tuicontext.ProgramContext) Model {
	keyMap := keys.Keys
	tabModel := tabs.NewModel(ctx)
	tabModel.SetTitles([]string{"Stars", "Trending", "Releases", "Stats"})

	footerModel := footer.NewModel(ctx)
	footerModel.SetLeft("q quit")
	footerModel.SetRight(ctx.Version)

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
		case key.Matches(typed, m.keys.Help):
			m.showHelp = !m.showHelp
		}
	}

	updated, cmd := m.stars.Update(msg)
	m.stars = updated
	m.syncSidebar()
	return m, cmd
}

func (m Model) View() tea.View {
	starsView := m.stars.View()
	if strings.TrimSpace(starsView) == "" {
		starsView = "No repos yet"
	}
	content := m.tabs.View() + "\n" + starsView
	if m.showSidebar {
		content += "\n" + m.sidebar.View()
	}
	if m.showHelp {
		content += "\n" + "j/k move  g/G first/last  p sidebar  ? help  q quit"
	}
	content += "\n" + m.footer.View()
	v := tea.NewView(content)
	v.AltScreen = true
	return v
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
