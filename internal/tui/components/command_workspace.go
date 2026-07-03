package components

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/tui/styles"
)

type CommandWorkspaceModel struct {
	theme    *styles.Theme
	selected *CommandNode
	params   map[string]string
	output   []string
	focus    int
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
	m.focus = 0
	for _, p := range node.Params {
		if p.DefaultValue != "" {
			m.params[p.Key] = p.DefaultValue
		}
	}
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

func (m *CommandWorkspaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *CommandWorkspaceModel) View() string {
	var b strings.Builder

	b.WriteString(m.renderDetail())
	b.WriteString(m.renderForm())
	b.WriteString(m.renderOutput())

	return b.String()
}

func (m *CommandWorkspaceModel) renderDetail() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Command Detail") + "\n")

	if m.selected == nil {
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Render("No command selected"))
		b.WriteString("\n\n")
		return b.String()
	}

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text)

	descStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	shortcutStyle := lipgloss.NewStyle().
		Foreground(m.theme.Secondary)

	b.WriteString(labelStyle.Render(m.selected.Label) + "\n")
	if m.selected.Description != "" {
		b.WriteString(descStyle.Render("  "+m.selected.Description) + "\n")
	}
	if m.selected.Shortcut != "" {
		b.WriteString(shortcutStyle.Render("  Shortcut: "+m.selected.Shortcut) + "\n")
	}

	b.WriteString("\n")
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
			Render("No parameters"))
		b.WriteString("\n\n")
		return b.String()
	}

	for i, p := range m.selected.Params {
		b.WriteString(m.renderParamField(p, i))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	return b.String()
}

func (m *CommandWorkspaceModel) renderParamField(p CommandParamSpec, index int) string {
	var b strings.Builder

	label := p.Label
	if p.Required {
		label = label + " *"
	}

	labelStyle := lipgloss.NewStyle().
		Width(14).
		Foreground(m.theme.Text)

	if index == m.focus {
		labelStyle = labelStyle.Foreground(m.theme.Primary).Bold(true)
	}

	value := m.params[p.Key]
	if value == "" {
		value = "(empty)"
	}

	valueStyle := lipgloss.NewStyle().
		Foreground(m.theme.Subtle)

	mutedStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted)

	b.WriteString("  ")
	b.WriteString(labelStyle.Render(label))
	b.WriteString(" ")
	b.WriteString(valueStyle.Render(value))
	b.WriteString(mutedStyle.Render("  ["+string(p.Type)+"]"))

	return b.String()
}

func (m *CommandWorkspaceModel) renderOutput() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Primary).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("Task / Output") + "\n")

	if len(m.output) == 0 {
		b.WriteString(lipgloss.NewStyle().
			Foreground(m.theme.Subtle).
			Render("No output yet"))
	} else {
		outputStyle := lipgloss.NewStyle().
			Foreground(m.theme.Subtle)
		for _, line := range m.output {
			b.WriteString(outputStyle.Render("  "+line) + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}
