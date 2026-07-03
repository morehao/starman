package releasessection

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/listviewport"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

const ShowUnread = "unread"
const ShowAll = "all"

var defaultColumns = []listviewport.Column{
	{Title: "repo", Width: 35},
	{Title: "version", Width: 16},
	{Title: "date", Width: 12},
	{Title: "status", Width: 7},
}

type GroupHeaderRow struct {
	Title string
}

func (r GroupHeaderRow) GetId() string       { return "group:" + r.Title }
func (r GroupHeaderRow) GetTitle() string    { return r.Title }
func (r GroupHeaderRow) GetUrl() string      { return "" }
func (r GroupHeaderRow) GetColumns() []string { return []string{} }

type ReleaseRow struct {
	Release *store.Release
}

func (r ReleaseRow) GetId() string {
	if r.Release == nil {
		return ""
	}
	return r.Release.TagName
}

func (r ReleaseRow) GetTitle() string {
	if r.Release == nil {
		return ""
	}
	return r.Release.RepoFullName + " " + r.Release.TagName
}

func (r ReleaseRow) GetUrl() string {
	if r.Release == nil {
		return ""
	}
	return r.Release.HTMLURL
}

func (r ReleaseRow) GetColumns() []string {
	if r.Release == nil {
		return []string{"", "", "", ""}
	}
	status := "unread"
	if r.Release.IsRead {
		status = "read"
	}
	date := r.Release.PublishedAt
	if len(date) > 10 {
		date = date[:10]
	}
	tag := r.Release.TagName
	if len(tag) > 16 {
		tag = tag[:15] + "…"
	}
	repo := r.Release.RepoFullName
	return []string{repo, tag, date, status}
}

type ReleasesFetchedMsg struct {
	SectionID int
	Releases  []*store.Release
	Err       error
}

type ReleasesMarkedMsg struct {
	SectionID int
	ReleaseID int64
}

type Model struct {
	id        int
	ctx       *tuicontext.ProgramContext
	cfg       section.SectionConfig
	filter    string
	list      listviewport.Model
	rows      []section.RowData
	loaded    bool
	isLoading bool
}

func NewModel(id int, ctx *tuicontext.ProgramContext, cfg section.SectionConfig, filter string) *Model {
	list := listviewport.New(defaultColumns, ctx)
	return &Model{id: id, ctx: ctx, cfg: cfg, filter: filter, list: list}
}

func (m *Model) GetId() int                                          { return m.id }
func (m *Model) GetType() string                                     { return "releases" }
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

func (m *Model) PrevRow()     { m.list.PrevRow() }
func (m *Model) FirstItem()   { m.list.FirstItem() }
func (m *Model) LastItem()    { m.list.LastItem() }

func (m *Model) SetFilter(filter string) {
	m.filter = filter
	m.ResetRows()
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	switch typed := msg.(type) {
	case ReleasesFetchedMsg:
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
		listRows := m.buildRows(typed.Releases)
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
		var releases []*store.Release
		var err error
		if m.filter == ShowAll {
			releases, err = m.ctx.Store.ListAllReleases(context.Background())
		} else {
			releases, err = m.ctx.Store.ListUnreadReleases(context.Background())
		}
		return ReleasesFetchedMsg{SectionID: m.id, Releases: releases, Err: err}
	}}
}

func (m *Model) MarkCurrentRead() tea.Cmd {
	row := m.CurrRow()
	if row == nil {
		return nil
	}
	if r, ok := row.(ReleaseRow); ok && r.Release != nil && !r.Release.IsRead {
		return func() tea.Msg {
			_ = m.ctx.Store.MarkReleaseRead(context.Background(), r.Release.ID)
			return ReleasesMarkedMsg{SectionID: m.id, ReleaseID: r.Release.ID}
		}
	}
	return nil
}

func (m *Model) MarkAllRead() tea.Cmd {
	if m.ctx == nil || m.ctx.Store == nil {
		return nil
	}
	return func() tea.Msg {
		_ = m.ctx.Store.MarkAllReleasesRead(context.Background())
		m.ResetRows()
		return ReleasesFetchedMsg{SectionID: m.id}
	}
}

func (m *Model) buildRows(releases []*store.Release) []section.RowData {
	m.rows = make([]section.RowData, 0, len(releases))
	rows := make([]section.RowData, 0, len(releases))
	prevRepo := ""
	for _, r := range releases {
		if r.RepoFullName != prevRepo {
			header := GroupHeaderRow{Title: r.RepoFullName}
			m.rows = append(m.rows, header)
			rows = append(rows, header)
			prevRepo = r.RepoFullName
		}
		row := ReleaseRow{Release: r}
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

func (m *Model) FilterLabel() string {
	if m.filter == ShowAll {
		return "All"
	}
	return "Unread"
}

func ReleaseSummary(r *store.Release) string {
	var b strings.Builder
	b.WriteString(r.TagName)
	if r.IsPrerelease {
		b.WriteString(" (pre-release)")
	}
	b.WriteString("\n")
	b.WriteString(r.PublishedAt)
	b.WriteString("\n\n")
	if r.Name != "" {
		b.WriteString(r.Name)
		b.WriteString("\n\n")
	}
	if r.Body != "" {
		body := r.Body
		if len(body) > 500 {
			body = body[:497] + "..."
		}
		b.WriteString(body)
	}
	return b.String()
}
