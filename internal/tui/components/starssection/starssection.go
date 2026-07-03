package starssection

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/listviewport"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

const (
	GroupAll = "all"
)

type RepoRow struct {
	Repo *store.Repository
}

func (r RepoRow) GetId() string {
	if r.Repo == nil {
		return ""
	}
	return r.Repo.FullName
}

func (r RepoRow) GetTitle() string {
	if r.Repo == nil {
		return ""
	}
	return r.Repo.FullName
}

func (r RepoRow) GetUrl() string {
	if r.Repo == nil {
		return ""
	}
	return r.Repo.URL
}

type ReposFetchedMsg struct {
	SectionID int
	Repos     []*store.Repository
}

type Model struct {
	id        int
	ctx       *tuicontext.ProgramContext
	cfg       section.SectionConfig
	groupBy   string
	list      listviewport.Model
	rows      []RepoRow
	loaded    bool
	isLoading bool
}

func NewModel(id int, ctx *tuicontext.ProgramContext, cfg section.SectionConfig, groupBy string) *Model {
	list := listviewport.New(nil, ctx)
	return &Model{id: id, ctx: ctx, cfg: cfg, groupBy: groupBy, list: list}
}

func (m *Model) GetId() int                        { return m.id }
func (m *Model) GetType() string                   { return "stars" }
func (m *Model) GetConfig() section.SectionConfig  { return m.cfg }
func (m *Model) NumRows() int                      { return len(m.rows) }
func (m *Model) CurrRowIndex() int                 { return m.list.Cursor() }
func (m *Model) View() string                      { return m.list.View() }
func (m *Model) GetIsLoading() bool                { return m.isLoading }
func (m *Model) GetTotalCount() int                { return len(m.rows) }
func (m *Model) IsSearchFocused() bool             { return false }
func (m *Model) ResetFilters()                     {}
func (m *Model) SetIsLoading(v bool)               { m.isLoading = v }
func (m *Model) UpdateProgramContext(ctx *tuicontext.ProgramContext) { m.ctx = ctx }

func (m *Model) CurrRow() section.RowData {
	idx := m.list.Cursor()
	if idx < 0 || idx >= len(m.rows) {
		return nil
	}
	return m.rows[idx]
}

func (m *Model) NextRow() section.RowData {
	m.list.NextRow()
	return m.CurrRow()
}

func (m *Model) PrevRow() {
	m.list.PrevRow()
}

func (m *Model) FirstItem() { m.list.FirstItem() }

func (m *Model) LastItem() { m.list.LastItem() }

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	switch typed := msg.(type) {
	case ReposFetchedMsg:
		if typed.SectionID != m.id {
			return m, nil
		}
		m.rows = make([]RepoRow, 0, len(typed.Repos))
		listRows := make([]section.RowData, 0, len(typed.Repos))
		for _, repo := range typed.Repos {
			row := RepoRow{Repo: repo}
			m.rows = append(m.rows, row)
			listRows = append(listRows, row)
		}
		m.list.SetRows(listRows)
		m.loaded = true
		m.isLoading = false
	}
	return m, nil
}

func (m *Model) FetchNextPageSectionRows() []tea.Cmd {
	if m.loaded || m.ctx == nil || m.ctx.Store == nil {
		return nil
	}
	m.isLoading = true
	return []tea.Cmd{func() tea.Msg {
		repos, _ := m.ctx.Store.ListRepositories(context.Background())
		return ReposFetchedMsg{SectionID: m.id, Repos: repos}
	}}
}

func (m *Model) ResetRows() {
	m.rows = nil
	m.list.SetRows(nil)
	m.loaded = false
	m.isLoading = false
}
