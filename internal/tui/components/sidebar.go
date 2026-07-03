package components

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type sidebarEntry struct {
	id       string
	label    string
	shortcut string
	page     types.PageID
}

type SidebarModel struct {
	theme       *styles.Theme
	cursor      int
	width       int
	height      int
	renderWidth int
	recentIDs   []string
	pinnedIDs   []string
	catalog     []CommandNode
	runningIDs  map[string]bool
}

func NewSidebar(theme *styles.Theme) *SidebarModel {
	return &SidebarModel{
		theme:       theme,
		cursor:      0,
		renderWidth: 30,
		catalog:     DefaultCommandCatalog(),
		pinnedIDs:   DefaultPinnedCommandIDs(),
	}
}

func (m *SidebarModel) SetRecent(ids []string) {
	m.recentIDs = append([]string(nil), ids...)
}

func (m *SidebarModel) SetRunning(id string, running bool) {
	if m.runningIDs == nil {
		m.runningIDs = make(map[string]bool)
	}
	if running {
		m.runningIDs[id] = true
	} else {
		delete(m.runningIDs, id)
	}
}

func (m *SidebarModel) SetRenderWidth(w int) {
	m.renderWidth = w
}

func (m *SidebarModel) Init() tea.Cmd { return nil }

func (m *SidebarModel) entries() []sidebarEntry {
	var entries []sidebarEntry
	for _, id := range m.recentIDs {
		if cmd, ok := resolveCommandID(m.catalog, id); ok {
			entries = append(entries, sidebarEntry{
				id: cmd.ID, label: cmd.Label, shortcut: cmd.Shortcut, page: cmd.Page,
			})
		}
	}
	for _, id := range m.pinnedIDs {
		if cmd, ok := resolveCommandID(m.catalog, id); ok {
			entries = append(entries, sidebarEntry{
				id: cmd.ID, label: cmd.Label, shortcut: cmd.Shortcut, page: cmd.Page,
			})
		}
	}
	return entries
}

func resolveCommandID(catalog []CommandNode, id string) (CommandNode, bool) {
	for _, n := range catalog {
		if n.ID == id {
			return n, true
		}
	}
	return CommandNode{}, false
}

func (m *SidebarModel) SelectedCommandID() (string, bool) {
	items := m.entries()
	if m.cursor >= 0 && m.cursor < len(items) {
		return items[m.cursor].id, true
	}
	return "", false
}

func (m *SidebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			total := len(m.entries())
			if total > 0 {
				m.cursor = min(m.cursor+1, total-1)
			}
		case "enter":
			items := m.entries()
			if m.cursor >= 0 && m.cursor < len(items) {
				item := items[m.cursor]
				if item.page >= 0 {
					return m, func() tea.Msg {
						return types.NavigatedMsg{Page: item.page}
					}
				}
			}
		}
	}
	return m, nil
}

func ShortcutPage(key string) (types.PageID, bool) {
	node, ok := FindByShortcut(DefaultCommandCatalog(), key)
	if !ok {
		return 0, false
	}
	return node.Page, true
}

func (m *SidebarModel) View() string {
	var s string

	labelW := m.renderWidth - 8
	if labelW < 8 {
		labelW = 8
	}

	idx := 0
	entries := m.entries()

	recentCount := 0
	for _, id := range m.recentIDs {
		if _, ok := resolveCommandID(m.catalog, id); ok {
			recentCount++
		}
	}

	if recentCount > 0 {
		s += m.theme.CardTitle.Render("RECENT") + "\n"
		for i, e := range entries {
			if i >= recentCount {
				break
			}
			status := "  "
			if m.runningIDs[e.id] {
				runningStyle := lipgloss.NewStyle().Foreground(m.theme.Success)
				status = runningStyle.Render(" ●")
			}
			line := fmt.Sprintf(" %2s %-*s%s", e.shortcut, labelW, e.label, status)
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

	s += m.theme.CardTitle.Render("PINNED") + "\n"
	for i, e := range entries {
		if i < recentCount {
			continue
		}
		line := fmt.Sprintf(" %2s %-*s", e.shortcut, labelW, e.label)
		if idx == m.cursor {
			s += m.theme.SidebarActive.Render(line)
		} else {
			s += m.theme.HelpText.Render(line)
		}
		s += "\n"
		idx++
	}
	s += "\n"

	return s
}
