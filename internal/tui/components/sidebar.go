package components

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui"
	"github.com/morehao/starman/internal/tui/styles"
)

type menuItem struct {
	label    string
	shortcut string
	page     tui.PageID
}

var menuItems = []menuItem{
	{"Dashboard", "1", tui.PageDashboard},
	{"Search", "/", tui.PageSearch},
	{"Repo List", "r", tui.PageRepoList},
	{"Trending", "t", tui.PageTrending},
	{"—", "", -1},
	{"Sync", "s", tui.PageSync},
	{"Analyze", "a", tui.PageAnalyze},
	{"—", "", -1},
	{"Tag", "g", tui.PageTag},
	{"Categorize", "c", tui.PageCategorize},
	{"Stats", "S", tui.PageStats},
	{"—", "", -1},
	{"Release", "R", tui.PageRelease},
	{"Generate", "G", tui.PageGenerate},
	{"Backup", "b", tui.PageBackup},
	{"Config", "C", tui.PageConfig},
	{"—", "", -1},
	{"Help", "?", tui.PageDashboard},
	{"Quit", "q", tui.PageDashboard},
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
		switch msg.String() {
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
				return m, func() tea.Msg { return tui.NavigatedMsg{Page: item.page} }
			}
		}
	}
	return m, nil
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
