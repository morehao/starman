package prompt

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

type PromptType int

const (
	PromptConfirm PromptType = iota
	PromptCategorySelect
	PromptTagEdit
	PromptCategoryForm
)

type PromptResultMsg struct {
	Type      PromptType
	Confirmed bool
	Value     string
}

type FormField struct {
	Label    string
	Value    string
	IsBool   bool
	Readonly bool
}

type Model struct {
	ptype   PromptType
	title   string
	options []string
	input   string
	cursor  int
	active  bool
	th      theme.Theme
	width   int

	formFields   []FormField
	formFieldIdx int
	formIsEdit   bool
}

func NewConfirmModel(title string) Model {
	return Model{ptype: PromptConfirm, title: title, active: true}
}

func NewCategorySelectModel(title string, categories []string, current string) Model {
	cursor := 0
	for i, c := range categories {
		if c == current {
			cursor = i
			break
		}
	}
	return Model{
		ptype:   PromptCategorySelect,
		title:   title,
		options: categories,
		cursor:  cursor,
		active:  true,
	}
}

func NewTagEditModel(title string, current string) Model {
	return Model{ptype: PromptTagEdit, title: title, input: current, active: true}
}

func NewCategoryFormModel(title string, fields []FormField, isEdit bool) Model {
	return Model{
		ptype:        PromptCategoryForm,
		title:        title,
		formFields:   fields,
		formFieldIdx: 0,
		active:       true,
		formIsEdit:   isEdit,
	}
}

func NewFormField(label, value string, isBool, readonly bool) FormField {
	return FormField{Label: label, Value: value, IsBool: isBool, Readonly: readonly}
}

func (m *Model) SetTheme(th theme.Theme) {
	m.th = th
}

func (m Model) IsFocused() bool { return m.active }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if !m.active {
			return m, nil
		}
		switch m.ptype {
		case PromptConfirm:
			return m.handleConfirmKeys(msg)
		case PromptCategorySelect:
			return m.handleCategoryKeys(msg)
		case PromptTagEdit:
			return m.handleTagKeys(msg)
		case PromptCategoryForm:
			return m.handleCategoryFormKeys(msg)
		}
	case tea.WindowSizeMsg:
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 27:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptConfirm, Confirmed: false} }
	case 'y', 'Y':
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptConfirm, Confirmed: true} }
	case 'n', 'N':
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptConfirm, Confirmed: false} }
	}
	return m, nil
}

func (m Model) handleCategoryKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 27:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptCategorySelect, Confirmed: false} }
	case 13:
		m.active = false
		return m, func() tea.Msg {
			return PromptResultMsg{Type: PromptCategorySelect, Confirmed: true, Value: m.options[m.cursor]}
		}
	case tea.KeyDown:
		if m.cursor < len(m.options)-1 {
			m.cursor++
		}
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m, nil
}

func (m Model) handleTagKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 27:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptTagEdit, Confirmed: false} }
	case 13:
		m.active = false
		return m, func() tea.Msg {
			return PromptResultMsg{Type: PromptTagEdit, Confirmed: true, Value: strings.TrimSpace(m.input)}
		}
	case 127:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if msg.Code >= 32 && msg.Code < 127 {
			m.input += string(rune(msg.Code))
		}
	}
	return m, nil
}

func (m Model) handleCategoryFormKeys(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 27:
		m.active = false
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptCategoryForm, Confirmed: false} }
	case 13:
		m.active = false
		values := make([]string, len(m.formFields))
		for i, f := range m.formFields {
			values[i] = f.Value
		}
		val := strings.Join(values, "\x00")
		return m, func() tea.Msg { return PromptResultMsg{Type: PromptCategoryForm, Confirmed: true, Value: val} }
	case tea.KeyDown:
		m.formFieldIdx++
		if m.formFieldIdx >= len(m.formFields) {
			m.formFieldIdx = 0
		}
	case tea.KeyUp:
		m.formFieldIdx--
		if m.formFieldIdx < 0 {
			m.formFieldIdx = len(m.formFields) - 1
		}
	case ' ':
		f := &m.formFields[m.formFieldIdx]
		if f.IsBool {
			if f.Value == "true" {
				f.Value = "false"
			} else {
				f.Value = "true"
			}
		}
	case 127:
		f := &m.formFields[m.formFieldIdx]
		if !f.IsBool && !f.Readonly && len(f.Value) > 0 {
			f.Value = f.Value[:len(f.Value)-1]
		}
	default:
		f := &m.formFields[m.formFieldIdx]
		if !f.IsBool && !f.Readonly && msg.Code >= 32 && msg.Code < 127 {
			f.Value += string(rune(msg.Code))
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if !m.active {
		return tea.NewView("")
	}

	overlayWidth := max(30, m.width/2)

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.th.FaintBorder).
		Padding(1, 2).
		Width(overlayWidth)

	var content string
	switch m.ptype {
	case PromptConfirm:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.title) + "\n" +
			lipgloss.NewStyle().Foreground(m.th.FaintText).Render("[y] yes  [n] no")
	case PromptCategorySelect:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.title) + "\n"
		for i, opt := range m.options {
			prefix := "  "
			line := prefix + opt
			if i == m.cursor {
				prefix = "> "
				line = prefix + opt
				content += lipgloss.NewStyle().
					Background(m.th.SelectedBackground).
					Foreground(m.th.PrimaryText).
					Render(line)
			} else {
				content += lipgloss.NewStyle().Foreground(m.th.FaintText).Render(line)
			}
			content += "\n"
		}
	case PromptTagEdit:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.title) + "\n" +
			lipgloss.NewStyle().Foreground(m.th.PrimaryText).Render(m.input) + "_"
	case PromptCategoryForm:
		content = lipgloss.NewStyle().Foreground(m.th.PrimaryText).Bold(true).Render(m.title)
		for i, f := range m.formFields {
			prefix := "  "
			linePrefix := prefix
			if m.formFieldIdx == i {
				linePrefix = "▸ "
			}

			if f.IsBool {
				toggleVal := "[ ]"
				if f.Value == "true" {
					toggleVal = "[x]"
				}
				fieldLine := linePrefix + f.Label + ": " + toggleVal
				if i == m.formFieldIdx {
					content += lipgloss.NewStyle().Background(m.th.SelectedBackground).Foreground(m.th.PrimaryText).Render(fieldLine)
				} else {
					content += lipgloss.NewStyle().Foreground(m.th.FaintText).Render(fieldLine)
				}
			} else {
				val := f.Value
				if f.Readonly {
					fieldLine := linePrefix + f.Label + ": " + val + " (只读)"
					if i == m.formFieldIdx {
						content += lipgloss.NewStyle().Background(m.th.SelectedBackground).Foreground(m.th.FaintText).Render(fieldLine)
					} else {
						content += lipgloss.NewStyle().Foreground(m.th.FaintText).Render(fieldLine)
					}
				} else {
					fieldLine := linePrefix + f.Label + ": " + val + "_"
					if i == m.formFieldIdx {
						content += lipgloss.NewStyle().Background(m.th.SelectedBackground).Foreground(m.th.PrimaryText).Render(fieldLine)
					} else {
						content += lipgloss.NewStyle().Foreground(m.th.FaintText).Render(fieldLine)
					}
				}
			}
			content += "\n"
		}
		content += "\n" + lipgloss.NewStyle().Foreground(m.th.FaintText).Render("[Enter] Save  [Esc] Cancel")
	}

	return tea.NewView(overlayStyle.Render(content))
}
