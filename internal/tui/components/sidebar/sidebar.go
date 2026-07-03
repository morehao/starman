package sidebar

import "github.com/morehao/starman/internal/tui/context"

type Model struct {
	ctx     *context.ProgramContext
	content string
}

func NewModel(ctx *context.ProgramContext) Model { return Model{ctx: ctx} }

func (m *Model) SetContent(content string) {
	m.content = content
}

func (m Model) View() string {
	if m.content == "" {
		return "Nothing selected..."
	}
	return m.content
}
