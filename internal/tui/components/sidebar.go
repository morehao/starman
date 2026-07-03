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

type menuGroup struct {
	title string
	items []menuItem
}

var groups = []menuGroup{
	{title: "发现", items: []menuItem{
		{"Search", "/", types.PageSearch},
		{"Trending", "t", types.PageTrending},
	}},
	{title: "整理", items: []menuItem{
		{"Repo List", "r", types.PageRepoList},
		{"Tag", "g", types.PageTag},
		{"Categorize", "c", types.PageCategorize},
		{"Stats", "S", types.PageStats},
	}},
	{title: "处理", items: []menuItem{
		{"Sync", "s", types.PageSync},
		{"Analyze", "a", types.PageAnalyze},
		{"Generate", "G", types.PageGenerate},
		{"Release", "R", types.PageRelease},
		{"Backup", "b", types.PageBackup},
	}},
	{title: "系统", items: []menuItem{
		{"Dashboard", "1", types.PageDashboard},
		{"Config", "C", types.PageConfig},
		{"Help", "?", types.PageDashboard},
		{"Quit", "q", types.PageDashboard},
	}},
}

func totalItems() int {
	n := 0
	for _, g := range groups {
		n += len(g.items)
	}
	return n
}

func itemAt(index int) *menuItem {
	for _, g := range groups {
		if index < len(g.items) {
			return &g.items[index]
		}
		index -= len(g.items)
	}
	return nil
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
		case "down", "j":
			m.cursor = min(m.cursor+1, totalItems()-1)
		case "enter":
			if item := itemAt(m.cursor); item != nil && item.label != "Quit" && item.page >= 0 {
				return m, func() tea.Msg { return types.NavigatedMsg{Page: item.page} }
			}
		}
	}
	return m, nil
}

func ShortcutPage(key string) (types.PageID, bool) {
	for _, g := range groups {
		for _, item := range g.items {
			if item.shortcut == key && item.page >= 0 {
				return item.page, true
			}
		}
	}
	return 0, false
}

func (m *SidebarModel) View() string {
	var s string
	s += m.theme.PageTitle.Render("STARMAN") + "\n\n"
	idx := 0
	for _, g := range groups {
		s += m.theme.CardTitle.Render(g.title) + "\n"
		for _, item := range g.items {
			line := fmt.Sprintf(" %-14s %2s", item.label, item.shortcut)
			if idx == m.cursor {
				s += m.theme.SidebarActive.Render(line)
			} else {
				s += m.theme.HelpText.Render(line)
			}
			s += "\n"
			idx++
		}
		s += "\n"
	}
	return s
}
