package trendingsection

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/discovery"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/listviewport"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

const (
	PeriodDaily   = "daily"
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
)

var defaultColumns = []listviewport.Column{
	{Title: "repo", Width: 28, Flex: true},
	{Title: "stars", Width: 8},
	{Title: "lang", Width: 14, Flex: true},
	{Title: "desc", Width: 30, Flex: true},
}

type TrendingRow struct {
	Repo *discovery.TrendingRepo
}

func (r TrendingRow) GetId() string {
	if r.Repo == nil {
		return ""
	}
	return r.Repo.FullName
}

func (r TrendingRow) GetTitle() string {
	if r.Repo == nil {
		return ""
	}
	return r.Repo.FullName
}

func (r TrendingRow) GetUrl() string {
	if r.Repo == nil {
		return ""
	}
	return r.Repo.URL
}

func (r TrendingRow) GetColumns() []string {
	if r.Repo == nil {
		return []string{"", "", "", ""}
	}
	stars := formatStarCount(r.Repo.Stars)
	lang := r.Repo.Language
	if lang == "" {
		lang = "-"
	}
	desc := r.Repo.Description
	if len(desc) > 30 {
		desc = desc[:29] + "…"
	}
	return []string{r.Repo.FullName, stars, lang, desc}
}

func TrendingToStoreRepo(tr *discovery.TrendingRepo) *store.Repository {
	return &store.Repository{
		FullName:        tr.FullName,
		Description:     tr.Description,
		URL:             tr.URL,
		Language:        tr.Language,
		StargazersCount: tr.Stars,
		ForksCount:      tr.Forks,
		Topics:          tr.Topics,
	}
}

type TrendingFetchedMsg struct {
	SectionID int
	Repos     []*discovery.TrendingRepo
	Err       error
}

type Model struct {
	id        int
	ctx       *tuicontext.ProgramContext
	cfg       section.SectionConfig
	period    string
	list      listviewport.Model
	rows      []section.RowData
	loaded    bool
	isLoading bool
}

func NewModel(id int, ctx *tuicontext.ProgramContext, cfg section.SectionConfig, period string) *Model {
	list := listviewport.New(defaultColumns, ctx)
	return &Model{id: id, ctx: ctx, cfg: cfg, period: period, list: list}
}

func (m *Model) GetId() int                                          { return m.id }
func (m *Model) GetType() string                                     { return "trending" }
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

func (m *Model) PrevRow() { m.list.PrevRow() }

func (m *Model) FirstItem() { m.list.FirstItem() }

func (m *Model) LastItem() { m.list.LastItem() }

func (m *Model) SetPeriod(period string) {
	if m.period == period {
		return
	}
	m.period = period
	m.loaded = false
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	switch typed := msg.(type) {
	case TrendingFetchedMsg:
		if typed.SectionID != m.id {
			return m, nil
		}
		if typed.Err != nil {
			m.rows = nil
			m.list.SetRows(nil)
			m.loaded = true
			m.isLoading = false
			return m, nil
		}
		listRows := m.buildRows(typed.Repos)
		m.list.SetRows(listRows)
		m.loaded = true
		m.isLoading = false
	}
	return m, nil
}

func (m *Model) FetchNextPageSectionRows() []tea.Cmd {
	if m.loaded || m.ctx == nil {
		return nil
	}
	m.isLoading = true
	return []tea.Cmd{func() tea.Msg {
		token := config.ResolveToken(m.ctx.Config, "")
		if token == "" {
			return TrendingFetchedMsg{SectionID: m.id, Err: fmt.Errorf("github token required")}
		}
		gh := github.New(token)
		svc := discovery.NewService(gh)
		repos, err := svc.Trending(context.Background(), discovery.TrendingOpts{
			Since:  m.period,
			Source: "rss",
		})
		return TrendingFetchedMsg{SectionID: m.id, Repos: repos, Err: err}
	}}
}

func (m *Model) buildRows(repos []*discovery.TrendingRepo) []section.RowData {
	m.rows = make([]section.RowData, 0, len(repos))
	rows := make([]section.RowData, 0, len(repos))
	for _, r := range repos {
		row := TrendingRow{Repo: r}
		m.rows = append(m.rows, row)
		rows = append(rows, row)
	}
	return rows
}

func (m *Model) ResetRows() {
	m.rows = nil
	m.list.SetRows(nil)
	m.loaded = false
	m.isLoading = false
}

func formatStarCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return strconv.Itoa(n)
}

func StringPeriod(s string) string {
	switch strings.ToLower(s) {
	case "daily":
		return "Daily"
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	default:
		return s
	}
}
