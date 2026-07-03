package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/tui/styles"
)

type CommandPaletteModel struct {
	theme    *styles.Theme
	catalog  []CommandNode
	filtered []CommandNode
	query    string
	cursor   int
	open     bool
}

func NewCommandPalette(theme *styles.Theme, catalog []CommandNode) *CommandPaletteModel {
	return &CommandPaletteModel{
		theme:   theme,
		catalog: catalog,
	}
}

func (m *CommandPaletteModel) Open() {
	m.open = true
	m.query = ""
	m.filtered = append([]CommandNode(nil), m.catalog...)
	m.cursor = 0
}

func (m *CommandPaletteModel) Close() {
	m.open = false
}

func (m *CommandPaletteModel) IsOpen() bool {
	return m.open
}

func (m *CommandPaletteModel) Selected() (CommandNode, bool) {
	if !m.open || len(m.filtered) == 0 || m.cursor < 0 || m.cursor >= len(m.filtered) {
		return CommandNode{}, false
	}
	return m.filtered[m.cursor], true
}

func (m *CommandPaletteModel) Update(msg tea.Msg) (*CommandPaletteModel, tea.Cmd) {
	if !m.open {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			m.query += string(msg.Runes)
			m.filtered = FilterCommands(m.catalog, m.query)
			m.cursor = 0
		case tea.KeyBackspace:
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				m.filtered = FilterCommands(m.catalog, m.query)
				m.cursor = 0
			}
		case tea.KeyEscape:
			m.Close()
		case tea.KeyEnter:
			return m, nil
		case tea.KeyTab:
			m.cursor = NextGroupIndex(m.filtered, m.cursor)
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}

func (m *CommandPaletteModel) View(width, height int) string {
	if !m.open {
		return ""
	}

	overlayWidth := min(width-8, width*3/4)
	if overlayWidth < 40 {
		overlayWidth = 40
	}
	innerW := overlayWidth - 4

	searchBar := m.renderSearchBar(innerW)
	groups := m.renderGroups(innerW)
	footer := m.renderFooter(innerW, len(m.catalog))

	title := "Command Palette"
	topBorder := "┌─ " + title + " ─" + strings.Repeat("─", overlayWidth-len(title)-6) + "┐"
	sep := "├" + strings.Repeat("─", overlayWidth-2) + "┤"
	bottomBorder := "└" + strings.Repeat("─", overlayWidth-2) + "┘"

	padToW := func(s string) string {
		w := lipgloss.Width(s)
		if w < innerW {
			s += strings.Repeat(" ", innerW-w)
		}
		return "│ " + s + " │"
	}

	contentLines := []string{
		topBorder,
		padToW(searchBar),
		sep,
	}

	groupLines := strings.Split(groups, "\n")
	for _, l := range groupLines {
		contentLines = append(contentLines, padToW(l))
	}

	contentLines = append(contentLines, padToW(footer))
	contentLines = append(contentLines, bottomBorder)

	content := strings.Join(contentLines, "\n")

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}


func (m *CommandPaletteModel) renderSearchBar(w int) string {
	prefix := "> " + m.query
	suffix := fmt.Sprintf("%d found", len(m.filtered))
	underscoreW := w - lipgloss.Width(prefix) - lipgloss.Width(suffix) - 2
	if underscoreW < 1 {
		underscoreW = 1
	}
	line := prefix + strings.Repeat("_", underscoreW) + "  " + suffix
	return lipgloss.NewStyle().Foreground(m.theme.Subtle).Render(line)
}

func (m *CommandPaletteModel) renderGroups(w int) string {
	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Render("No commands found")
	}

	var groups []string
	var currentGroup string
	var groupStart int

	for i, n := range m.filtered {
		if n.Group != currentGroup {
			if currentGroup != "" {
				groups = append(groups, m.renderGroup(currentGroup, m.filtered[groupStart:i], groupStart, w))
			}
			currentGroup = n.Group
			groupStart = i
		}
	}
	groups = append(groups, m.renderGroup(currentGroup, m.filtered[groupStart:], groupStart, w))

	return strings.Join(groups, "\n")
}

func (m *CommandPaletteModel) renderGroup(name string, nodes []CommandNode, offset, w int) string {
	var b strings.Builder

	groupStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary)
	b.WriteString(groupStyle.Render(strings.ToUpper(name)) + "\n")

	normalStyle := lipgloss.NewStyle().Foreground(m.theme.Subtle)
	selectedStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(m.theme.Primary)

	for i, n := range nodes {
		idx := offset + i
		prefix := "  "
		if idx == m.cursor {
			prefix = cursorStyle.Render(" ▸")
		}
		line := fmt.Sprintf("%s %-2s %-16s %s", prefix, n.Shortcut, n.Label, n.Description)
		if idx == m.cursor {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	return b.String()
}

func (m *CommandPaletteModel) renderFooter(w int, total int) string {
	footerStyle := lipgloss.NewStyle().Foreground(m.theme.Subtle)
	if len(m.filtered) == 0 {
		return footerStyle.Render("Esc 关闭")
	}
	nav := "↑↓ 选择   Enter 进入   Tab 切组   Esc 关闭"
	count := fmt.Sprintf("(%d/%d 全部)", m.cursor+1, total)
	navW := lipgloss.Width(nav)
	countW := lipgloss.Width(count)
	padding := w - navW - countW - 2
	if padding < 2 {
		padding = 2
	}
	return footerStyle.Render(nav + strings.Repeat(" ", padding) + count)
}
