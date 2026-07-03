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

	overlayWidth := min(width-10, 64)
	overlayHeight := min(height-4, 20)

	var b strings.Builder

	b.WriteString(m.renderHeader(overlayWidth) + "\n")
	b.WriteString(m.renderSearchBar(overlayWidth) + "\n")
	b.WriteString(m.renderGroups(overlayWidth, overlayHeight) + "\n")
	b.WriteString(m.renderFooter(overlayWidth))

	overlayStyle := lipgloss.NewStyle().
		Width(overlayWidth).
		Height(overlayHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary).
		Padding(1)

	return lipgloss.Place(width, height,
		lipgloss.Center, lipgloss.Center,
		overlayStyle.Render(b.String()),
	)
}

func (m *CommandPaletteModel) renderHeader(w int) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		Width(w - 4).
		Render("Command Palette")
}

func (m *CommandPaletteModel) renderSearchBar(w int) string {
	display := "> " + m.query
	cursor := " "
	if m.query == "" {
		cursor = "|"
	}
	return lipgloss.NewStyle().
		Foreground(m.theme.Text).
		Width(w - 4).
		Render(display + cursor)
}

func (m *CommandPaletteModel) renderGroups(w, maxH int) string {
	if len(m.filtered) == 0 {
		return lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Width(w - 4).
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

	separator := strings.Repeat("─", w-6)
	b.WriteString(lipgloss.NewStyle().
		Foreground(m.theme.Subtle).
		Render("─ " + name + " " + separator[len(name)+4:]) + "\n")

	for i, n := range nodes {
		idx := offset + i
		line := fmt.Sprintf("  %-16s %-4s  %s", n.Label, n.Shortcut, n.Description)
		lineStyle := lipgloss.NewStyle().Foreground(m.theme.Subtle)
		if idx == m.cursor {
			lineStyle = lipgloss.NewStyle().
				Foreground(m.theme.Text).
				Background(m.theme.Primary).
				Bold(true)
		}
		b.WriteString(lineStyle.Width(w - 6).Render(line) + "\n")
	}

	return b.String()
}

func (m *CommandPaletteModel) renderFooter(w int) string {
	count := len(m.filtered)
	if count == 0 {
		return lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Width(w - 4).
			Render("Press Esc to close")
	}
	return lipgloss.NewStyle().
		Foreground(m.theme.Subtle).
		Width(w - 4).
		Render(fmt.Sprintf("%d commands found  ·  Tab: next group  ·  Esc: close", count))
}
