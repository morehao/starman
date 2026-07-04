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
	taskStatus string
	task       *TaskInfo
}

func NewModel(ctx *context.ProgramContext) Model {
	return Model{ctx: ctx}
}

func (m *Model) SetPager(v string)      { m.pager = v }
func (m *Model) SetTask(t *TaskInfo)     { m.task = t }

func (m Model) View() string {
	theme := m.ctx.Theme
	activeStyle := lipgloss.NewStyle().
		Foreground(theme.PrimaryText).Bold(true)
	inactiveStyle := lipgloss.NewStyle().
		Foreground(theme.FaintText)
	bgStyle := lipgloss.NewStyle().
		Background(theme.SelectedBackground).
		Width(m.ctx.ScreenWidth)
	faintStyle := lipgloss.NewStyle().
		Foreground(theme.FaintText).
		Background(theme.SelectedBackground)
	successStyle := lipgloss.NewStyle().
		Foreground(theme.SuccessText).
		Background(theme.SelectedBackground)
	errorStyle := lipgloss.NewStyle().
		Foreground(theme.ErrorText).
		Background(theme.SelectedBackground)

	views := []struct {
		icon  string
		label string
	}{
		{"⭐", "Stars"},
		{"📈", "Trending"},
		{"📦", "Releases"},
		{"📊", "Stats"},
	}

	var viewParts []string
	for i, v := range views {
		label := v.icon + v.label
		if string(m.ctx.View) == getViewKey(i) {
			viewParts = append(viewParts, activeStyle.Render(label))
		} else {
			viewParts = append(viewParts, inactiveStyle.Render(label))
		}
	}
	viewSwitcher := strings.Join(viewParts, " | ")

	var rightParts []string

	if m.task != nil {
		switch m.task.Status {
		case 0:
			frame := spinnerFrames[m.task.SpinnerIdx%len(spinnerFrames)]
			rightParts = append(rightParts, frame+" "+m.task.Message)
		case 1:
			rightParts = append(rightParts, successStyle.Render("✅ "+m.task.Message))
		case 2:
			msg := m.task.Message
			if m.task.Err != nil {
				msg = m.task.Message + ": " + m.task.Err.Error()
			}
			rightParts = append(rightParts, errorStyle.Render("❌ "+msg))
		}
	}

	if m.pager != "" {
		rightParts = append(rightParts, m.pager)
	}
	rightParts = append(rightParts, faintStyle.Render("?help"))

	rightStr := strings.Join(rightParts, "  ")

	return bgStyle.Render(lipgloss.JoinHorizontal(
		lipgloss.Top,
		lipgloss.NewStyle().Background(theme.SelectedBackground).Render(viewSwitcher),
		lipgloss.NewStyle().
			Background(theme.SelectedBackground).
			Width(m.ctx.ScreenWidth-lipgloss.Width(viewSwitcher)-lipgloss.Width(rightStr)).
			Render(""),
		lipgloss.NewStyle().Background(theme.SelectedBackground).Render(rightStr),
	))
}

func getViewKey(i int) string {
	switch i {
	case 0:
		return string(context.StarsView)
	case 1:
		return string(context.TrendingView)
	case 2:
		return string(context.ReleasesView)
	case 3:
		return string(context.StatsView)
	default:
		return ""
	}
}

func (m Model) ViewSwitcherAtX(x int) int {
	views := []string{"⭐Stars", "📈Trending", "📦Releases", "📊Stats"}
	offset := 0
	for i, label := range views {
		w := offset + len(label) + 3
		if x >= offset && x < w {
			return i
		}
		offset = w
	}
	return -1
}
