package footer

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type Model struct {
	ctx        *context.ProgramContext
	pager      string
	taskStatus string
}

func NewModel(ctx *context.ProgramContext) Model {
	return Model{ctx: ctx}
}

func (m *Model) SetPager(v string)      { m.pager = v }
func (m *Model) SetTaskStatus(v string) { m.taskStatus = v }

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
	if m.pager != "" {
		rightParts = append(rightParts, m.pager)
	}
	if m.taskStatus != "" {
		rightParts = append(rightParts, m.taskStatus)
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
