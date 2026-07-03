package tabs

import (
	"strings"

	"github.com/morehao/starman/internal/tui/context"
)

type Model struct {
	ctx    *context.ProgramContext
	titles []string
}

func NewModel(ctx *context.ProgramContext) Model { return Model{ctx: ctx} }

func (m *Model) SetTitles(titles []string) {
	m.titles = titles
}

func (m Model) View() string {
	return strings.Join(m.titles, " | ")
}
