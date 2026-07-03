package footer

import (
	"charm.land/lipgloss/v2"

	"github.com/morehao/starman/internal/tui/context"
)

type Model struct {
	ctx   *context.ProgramContext
	left  string
	right string
}

func NewModel(ctx *context.ProgramContext) Model { return Model{ctx: ctx} }

func (m Model) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.left, " ", m.right, " ", "?help")
}

func (m *Model) SetLeft(v string)  { m.left = v }
func (m *Model) SetRight(v string) { m.right = v }
