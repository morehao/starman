package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

type placeholderSection struct {
	id     int
	kind   string
	title  string
	ctx    *tuicontext.ProgramContext
}

func newPlaceholderSection(id int, kind, title string, ctx *tuicontext.ProgramContext) *placeholderSection {
	return &placeholderSection{id: id, kind: kind, title: title, ctx: ctx}
}

func (p *placeholderSection) GetId() int                               { return p.id }
func (p *placeholderSection) GetType() string                          { return p.kind }
func (p *placeholderSection) GetConfig() section.SectionConfig         { return section.SectionConfig{Title: p.title} }
func (p *placeholderSection) CurrRow() section.RowData                 { return nil }
func (p *placeholderSection) NextRow() section.RowData                 { return nil }
func (p *placeholderSection) PrevRow()                                 {}
func (p *placeholderSection) FirstItem()                               {}
func (p *placeholderSection) LastItem()                                {}
func (p *placeholderSection) NumRows() int                             { return 0 }
func (p *placeholderSection) CurrRowIndex() int                        { return 0 }
func (p *placeholderSection) SetSize(w, h int)                         {}
func (p *placeholderSection) Pager() string                            { return "" }
func (p *placeholderSection) View() string {
	theme := p.ctx.Theme
	dimStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	boldStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)

	w := p.ctx.MainContentWidth
	if w < 30 {
		w = 30
	}
	h := p.ctx.MainContentHeight
	if h < 3 {
		h = 3
	}

	var lines []string
	for i := 0; i < h/2-1; i++ {
		lines = append(lines, "")
	}
	lines = append(lines, centerText(boldStyle.Render(p.title), w))
	lines = append(lines, centerText(dimStyle.Render("Coming soon..."), w))
	for i := len(lines); i < h; i++ {
		lines = append(lines, "")
	}
	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Render(strings.Join(lines, "\n"))
}
func (p *placeholderSection) Update(msg tea.Msg) (section.Section, tea.Cmd) { return p, nil }
func (p *placeholderSection) FetchNextPageSectionRows() []tea.Cmd            { return nil }
func (p *placeholderSection) ResetFilters()                                   {}
func (p *placeholderSection) ResetRows()                                      {}
func (p *placeholderSection) SetIsLoading(v bool)                             {}
func (p *placeholderSection) GetIsLoading() bool                              { return false }
func (p *placeholderSection) GetTotalCount() int                              { return 0 }
func (p *placeholderSection) IsSearchFocused() bool                           { return false }
func (p *placeholderSection) UpdateProgramContext(ctx *tuicontext.ProgramContext) { p.ctx = ctx }
func (p *placeholderSection) FilterRows(query string)                               {}
func (p *placeholderSection) SupportsSearch() bool                                  { return false }
func (p *placeholderSection) SupportsFilter() bool                                  { return false }
