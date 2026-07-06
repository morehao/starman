package footer

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type TaskInfo struct {
	Status     int
	Message    string
	Err        error
	SpinnerIdx int
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type Model struct {
	ctx        *context.ProgramContext
	pager      string
	task       *TaskInfo
}

func NewModel(ctx *context.ProgramContext) Model {
	return Model{ctx: ctx}
}

func (m *Model) SetPager(v string)      { m.pager = v }
func (m *Model) SetTask(t *TaskInfo)     { m.task = t }

func (m Model) View() string {
	theme := m.ctx.Theme
	bgStyle := lipgloss.NewStyle().
		Width(m.ctx.ScreenWidth).
		Align(lipgloss.Right).
		Background(theme.SelectedBackground)

	helpText := "j/k move │ g/G first/last │ p sidebar │ m action menu │ / search │ : cmd │ q quit"
	help := lipgloss.NewStyle().
		Foreground(theme.FaintText).
		Background(theme.SelectedBackground).
		Render(helpText)

	var rightParts []string

	if m.task != nil {
		switch m.task.Status {
		case 0:
			frame := spinnerFrames[m.task.SpinnerIdx%len(spinnerFrames)]
			rightParts = append(rightParts, frame+" "+m.task.Message)
		case 1:
			rightParts = append(rightParts,
				lipgloss.NewStyle().
					Foreground(theme.SuccessText).
					Background(theme.SelectedBackground).
					Render("✅ "+m.task.Message))
		case 2:
			msg := m.task.Message
			if m.task.Err != nil {
				msg = m.task.Message + ": " + m.task.Err.Error()
			}
			rightParts = append(rightParts,
				lipgloss.NewStyle().
					Foreground(theme.ErrorText).
					Background(theme.SelectedBackground).
					Render("❌ "+msg))
		}
	}

	if m.pager != "" {
		rightParts = append(rightParts, m.pager)
	}

	var parts []string
	parts = append(parts, help)
	if len(rightParts) > 0 {
		parts = append(parts, "│", strings.Join(rightParts, "  "))
	}

	return bgStyle.Render(strings.Join(parts, "  "))
}


