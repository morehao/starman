package components

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type menuItem struct {
	label    string
	shortcut string
	page     types.PageID
}

var menuItems = []menuItem{
	{"Dashboard", "1", types.PageDashboard},
	{"Search", "/", types.PageSearch},
	{"Repo List", "r", types.PageRepoList},
	{"Trending", "t", types.PageTrending},
	{"—", "", -1},
	{"Sync", "s", types.PageSync},
	{"Analyze", "a", types.PageAnalyze},
	{"—", "", -1},
	{"Tag", "g", types.PageTag},
	{"Categorize", "c", types.PageCategorize},
	{"Stats", "S", types.PageStats},
	{"—", "", -1},
	{"Release", "R", types.PageRelease},
	{"Generate", "G", types.PageGenerate},
	{"Backup", "b", types.PageBackup},
	{"Config", "C", types.PageConfig},
	{"—", "", -1},
	{"Help", "?", types.PageDashboard},
	{"Quit", "q", types.PageDashboard},
}

type SidebarModel struct {
	theme  *styles.Theme
	cursor int
	width  int
	height int
}

func NewSidebar(theme *styles.Theme) *SidebarModel {
	return &SidebarModel{
		theme:  theme,
		cursor: 0,
	}
}

func (m *SidebarModel) Init() tea.Cmd { return nil }

func (m *SidebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = m.theme.Sidebar.GetWidth()
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
			if menuItems[m.cursor].page == -1 {
				m.cursor = max(m.cursor-1, 0)
			}
		case "down", "j":
			m.cursor = min(m.cursor+1, len(menuItems)-1)
			if menuItems[m.cursor].page == -1 {
				m.cursor = min(m.cursor+1, len(menuItems)-1)
			}
		case "enter":
			if item := menuItems[m.cursor]; item.page >= 0 && item.label != "Quit" {
				return m, func() tea.Msg { return types.NavigatedMsg{Page: item.page} }
			}
		}
	}
	return m, nil
}

func ShortcutPage(key string) (types.PageID, bool) {
	for i := range menuItems {
		if menuItems[i].shortcut == key && menuItems[i].page >= 0 {
			return menuItems[i].page, true
		}
	}
	return 0, false
}

func (m *SidebarModel) View() string {
	var s string
	s += m.theme.PageTitle.Render("STARMAN") + "\n\n"
	for i, item := range menuItems {
		if item.label == "—" {
			s += "\n"
			continue
		}
		line := fmt.Sprintf(" %-14s %2s", item.label, item.shortcut)
		if i == m.cursor {
			s += m.theme.SidebarActive.Render(line)
		} else {
			s += m.theme.HelpText.Render(line)
		}
		s += "\n"
	}
	return s
}
