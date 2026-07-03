package components

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type CommandWorkspaceModel struct {
	theme    *styles.Theme
	selected *CommandNode
	params   map[string]string
	output   []string
	width    int
	height   int
}

func NewCommandWorkspace(theme *styles.Theme) *CommandWorkspaceModel {
	return &CommandWorkspaceModel{
		theme:  theme,
		params: make(map[string]string),
	}
}

func (m *CommandWorkspaceModel) SelectCommand(node CommandNode) {
	m.selected = &node
	m.params = make(map[string]string)
	m.output = nil
	for _, p := range node.Params {
		if p.DefaultValue != "" {
			m.params[p.Key] = p.DefaultValue
		}
	}
}

func (m *CommandWorkspaceModel) SelectedID() string {
	if m.selected == nil {
		return ""
	}
	return m.selected.ID
}

func (m *CommandWorkspaceModel) Params() map[string]string {
	return m.params
}

func (m *CommandWorkspaceModel) AppendOutput(line string) {
	m.output = append(m.output, line)
}

func (m *CommandWorkspaceModel) Validate() error {
	if m.selected == nil {
		return errors.New("no command selected")
	}
	for _, p := range m.selected.Params {
		if p.Required && strings.TrimSpace(m.params[p.Key]) == "" {
			return fmt.Errorf("%s is required", p.Label)
		}
	}
	return nil
}

func (m *CommandWorkspaceModel) Init() tea.Cmd { return nil }

func (m *CommandWorkspaceModel) contentWidth() int {
	cw := m.width*7/10 - 5
	if cw < 30 {
		cw = 30
	}
	return cw
}

func (m *CommandWorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case types.TaskProgressMsg:
		pct := 100
		if msg.Total > 0 {
			pct = msg.Current * 100 / msg.Total
		}
		bar := fmt.Sprintf("  ██████████████████░░░░░░░░░░░░░░  %d%%", pct)
		if len(m.output) > 0 && strings.HasPrefix(m.output[len(m.output)-1], "  ██") {
			m.output[len(m.output)-1] = bar
		} else {
			m.output = append(m.output, bar)
		}
	}
	return m, nil
}

func (m *CommandWorkspaceModel) View() string {
	var b strings.Builder
	b.WriteString(m.renderDetail())
	if m.selected != nil && len(m.selected.Params) > 0 {
		b.WriteString(m.sectionSep())
		b.WriteString(m.renderForm())
	}
	b.WriteString(m.sectionSep())
	b.WriteString(m.renderOutput())
	return b.String()
}

func (m *CommandWorkspaceModel) sectionSep() string {
	sepStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)
	innerW := m.width - 35
	if innerW < 10 {
		innerW = 10
	}
	return sepStyle.Render("\n" + strings.Repeat("─", innerW) + "\n")
}

func (m *CommandWorkspaceModel) renderDetail() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text)

	descStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	shortcutStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary)

	if m.selected == nil {
		b.WriteString(titleStyle.Render("Command Detail") + "\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Render("  Ctrl+K to select a command"))
		b.WriteString("\n")
		return b.String()
	}

	innerW := m.contentWidth()

	title := labelStyle.Render("▍" + m.selected.Label)
	shortcut := shortcutStyle.Render("[" + m.selected.Shortcut + "]")
	titleW := lipgloss.Width(title)
	shortcutW := lipgloss.Width(shortcut)
	padding := innerW - titleW - shortcutW - 2
	if padding < 1 {
		padding = 1
	}

	b.WriteString(title)
	b.WriteString(strings.Repeat(" ", padding))
	b.WriteString(shortcut)
	b.WriteString("\n")
	if m.selected.Description != "" {
		b.WriteString(descStyle.Render("  " + m.selected.Description) + "\n")
	}
	return b.String()
}

func (m *CommandWorkspaceModel) renderForm() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Params") + "\n")

	if m.selected == nil || len(m.selected.Params) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Render("  No parameters"))
		return b.String()
	}

	for _, p := range m.selected.Params {
		b.WriteString(m.renderParamField(p))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *CommandWorkspaceModel) renderParamField(p CommandParamSpec) string {
	var b strings.Builder

	label := p.Label
	if p.Required {
		label = label + " *"
	}

	labelStyle := lipgloss.NewStyle().
		Foreground(m.theme.Primary).
		Bold(true)

	value := m.params[p.Key]
	if value == "" {
		value = "(empty)"
	}

	valueStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	mutedStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	b.WriteString("  ")
	b.WriteString(labelStyle.Render(label) + ":")
	b.WriteString(valueStyle.Render("["+value+"]"))
	b.WriteString(mutedStyle.Render(" ["+string(p.Type)+"]"))

	return b.String()
}

func (m *CommandWorkspaceModel) renderOutput() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		MarginBottom(1)

	scrollStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	outputStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	timestampStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	b.WriteString(titleStyle.Render("Output") + scrollStyle.Render("  ▲ ▼") + "\n")

	if len(m.output) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Render("  No output yet"))
		return b.String()
	}

	now := time.Now().Format("15:04:05")

	for i, line := range m.output {
		prefix := timestampStyle.Render("[" + now + "] ")

		if strings.HasPrefix(line, "✓") {
			prefix += lipgloss.NewStyle().Foreground(m.theme.Success).Render(line)
		} else if strings.HasPrefix(line, "✗") {
			prefix += lipgloss.NewStyle().Foreground(m.theme.Error).Render(line)
		} else {
			prefix += outputStyle.Render(line)
		}

		if i == 0 {
			b.WriteString(prefix + "\n")
		} else {
			b.WriteString(prefix + "\n")
		}
	}

	return b.String()
}
