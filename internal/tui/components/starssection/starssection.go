package starssection

import (
	"context"
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/listviewport"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

const (
	GroupAll      = "all"
	GroupLanguage = "language"
	GroupCategory = "category"
	GroupTag      = "tag"
)

var defaultColumns = []listviewport.Column{
	{Title: "repo", Width: 40},
	{Title: "stars", Width: 7},
	{Title: "lang", Width: 12},
	{Title: "cat", Width: 12},
}

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

func (r RepoRow) GetColumns() []string {
	if r.Repo == nil {
		return []string{"", "", "", ""}
	}
	stars := fmt.Sprintf("%d", r.Repo.StargazersCount)
	lang := r.Repo.Language
	if lang == "" {
		lang = "—"
	}
	if len(lang) > 12 {
		lang = lang[:11] + "…"
	}
	cat := r.Repo.AICategory
	if cat == "" {
		cat = r.Repo.CustomCategory
	}
	if cat == "" {
		cat = "—"
	}
	return []string{r.Repo.FullName, stars, lang, cat}
}

type ReposFetchedMsg struct {
	SectionID int
	Repos     []*store.Repository
}

type ReposFetchFailedMsg struct {
	SectionID int
	Err       error
}

type GroupHeaderRow struct {
	Title string
}

func (r GroupHeaderRow) GetId() string      { return "group:" + r.Title }
func (r GroupHeaderRow) GetTitle() string   { return r.Title }
func (r GroupHeaderRow) GetUrl() string     { return "" }
func (r GroupHeaderRow) GetColumns() []string { return []string{} }

type Model struct {
	id        int
	ctx       *tuicontext.ProgramContext
	cfg       section.SectionConfig
	groupBy   string
	list      listviewport.Model
	rows      []section.RowData
	loaded    bool
	isLoading bool
	groupData *GroupedRepos
}

func NewModel(id int, ctx *tuicontext.ProgramContext, cfg section.SectionConfig, groupBy string) *Model {
	list := listviewport.New(defaultColumns, ctx)
	return &Model{id: id, ctx: ctx, cfg: cfg, groupBy: groupBy, list: list}
}

func (m *Model) GetId() int                                          { return m.id }
func (m *Model) GetType() string                                     { return "stars" }
func (m *Model) GetConfig() section.SectionConfig                    { return m.cfg }
func (m *Model) NumRows() int                                        { return len(m.rows) }
func (m *Model) CurrRowIndex() int                                   { return m.list.Cursor() }
func (m *Model) View() string                                        { return m.list.View() }
func (m *Model) GetIsLoading() bool                                  { return m.isLoading }
func (m *Model) GetTotalCount() int                                  { return len(m.rows) }
func (m *Model) IsSearchFocused() bool                               { return false }
func (m *Model) ResetFilters()                                       {}
func (m *Model) SetIsLoading(v bool)                                 { m.isLoading = v }
func (m *Model) UpdateProgramContext(ctx *tuicontext.ProgramContext) { m.ctx = ctx }

func (m *Model) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

func (m *Model) Pager() string {
	return m.list.Pager()
}

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

func (m *Model) GroupCounts() map[string]int {
	if m.groupData == nil {
		return nil
	}
	counts := make(map[string]int)
	for _, key := range m.groupData.KeysSorted() {
		counts[key] = m.groupData.BucketCount(key)
	}
	return counts
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	switch typed := msg.(type) {
	case ReposFetchedMsg:
		if typed.SectionID != m.id {
			return m, nil
		}
		listRows := m.buildRows(typed.Repos)
		m.list.SetRows(listRows)
		m.loaded = true
		m.isLoading = false
	case ReposFetchFailedMsg:
		if typed.SectionID != m.id {
			return m, nil
		}
		m.rows = nil
		m.list.SetRows(nil)
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
		repos, err := m.ctx.Store.ListRepositories(context.Background())
		if err != nil {
			return ReposFetchFailedMsg{SectionID: m.id, Err: err}
		}
		return ReposFetchedMsg{SectionID: m.id, Repos: repos}
	}}
}

func (m *Model) buildRows(repos []*store.Repository) []section.RowData {
	if m.groupBy == GroupAll {
		m.rows = make([]section.RowData, 0, len(repos))
		rows := make([]section.RowData, 0, len(repos))
		for _, repo := range repos {
			repoRow := RepoRow{Repo: repo}
			m.rows = append(m.rows, repoRow)
			rows = append(rows, repoRow)
		}
		return rows
	}

	grouped := groupReposBy(m.groupBy, repos)
	m.groupData = &grouped
	keys := grouped.KeysSorted()
	m.rows = make([]section.RowData, 0, len(repos)+len(keys))
	rows := make([]section.RowData, 0, len(repos)+len(keys))
	for _, key := range keys {
		header := GroupHeaderRow{Title: fmt.Sprintf("%s (%d)", key, grouped.BucketCount(key))}
		m.rows = append(m.rows, header)
		rows = append(rows, header)
		for _, repo := range grouped.Get(key) {
			repoRow := RepoRow{Repo: repo}
			m.rows = append(m.rows, repoRow)
			rows = append(rows, repoRow)
		}
	}
	return rows
}

func groupReposBy(groupBy string, repos []*store.Repository) GroupedRepos {
	switch groupBy {
	case GroupLanguage:
		return GroupByLanguage(repos)
	case GroupCategory:
		return GroupByCategory(repos)
	case GroupTag:
		return GroupByTag(repos)
	default:
		return GroupByLanguage(repos)
	}
}

func (m *Model) SetGroupBy(groupBy string) {
	if m.groupBy != groupBy {
		m.groupBy = groupBy
		m.ResetRows()
	}
}

func (m *Model) ResetRows() {
	m.rows = nil
	m.list.SetRows(nil)
	m.loaded = false
	m.isLoading = false
	m.groupData = nil
}

func formatStarCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return strconv.Itoa(n)
}
