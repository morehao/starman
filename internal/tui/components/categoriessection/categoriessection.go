package categoriessection

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/listviewport"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

var defaultColumns = []listviewport.Column{
	{Title: "ID", Width: 14},
	{Title: "Name", Width: 16},
	{Title: "Keywords", Width: 20, Flex: true},
	{Title: "Repos", Width: 7},
	{Title: "Sort", Width: 5},
	{Title: "Type", Width: 8},
}

type CategoryRow struct {
	Category  *store.Category
	RepoCount int
}

func (r CategoryRow) GetId() string    { return r.Category.ID }
func (r CategoryRow) GetTitle() string { return r.Category.Name }
func (r CategoryRow) GetUrl() string   { return "" }

func (r CategoryRow) GetColumns() []string {
	kwStr := ""
	if len(r.Category.Keywords) > 0 {
		kwStr = joinKeywords(r.Category.Keywords)
	}
	typeStr := "内置"
	if r.Category.IsCustom {
		typeStr = "自定义"
	}
	return []string{
		r.Category.ID,
		r.Category.Name,
		kwStr,
		itoa(r.RepoCount),
		itoa(r.Category.SortOrder),
		typeStr,
	}
}

type Model struct {
	id         int
	ctx        *tuicontext.ProgramContext
	cfg        section.SectionConfig
	list       listviewport.Model
	categories []*store.Category
	rows       []section.RowData
	loaded     bool
	isLoading  bool
}

type CategoriesFetchedMsg struct {
	SectionID  int
	Categories []*store.Category
	Repos      []*store.Repository
	Err        error
}

func NewModel(id int, ctx *tuicontext.ProgramContext, cfg section.SectionConfig) *Model {
	list := listviewport.New(defaultColumns, ctx)
	return &Model{id: id, ctx: ctx, cfg: cfg, list: list}
}

func (m *Model) GetId() int                                          { return m.id }
func (m *Model) GetType() string                                     { return "categories" }
func (m *Model) GetConfig() section.SectionConfig                    { return m.cfg }
func (m *Model) GetTotalCount() int                                  { return len(m.rows) }
func (m *Model) GetIsLoading() bool                                  { return m.isLoading }
func (m *Model) SetIsLoading(v bool)                                 { m.isLoading = v }
func (m *Model) IsSearchFocused() bool                               { return false }
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

func (m *Model) FirstItem() {
	m.list.FirstItem()
}

func (m *Model) LastItem() {
	m.list.LastItem()
}

func (m *Model) NumRows() int {
	return len(m.rows)
}

func (m *Model) CurrRowIndex() int {
	return m.list.Cursor()
}

func (m *Model) ResetRows() {
	m.loaded = false
	m.rows = nil
	m.list.SetRows(nil)
}

func (m *Model) ResetFilters() {
	if !m.loaded {
		return
	}
	m.rows = m.buildRows(m.categories, nil)
	m.list.SetRows(m.rows)
}

func (m *Model) FetchNextPageSectionRows() []tea.Cmd {
	if m.loaded || m.ctx == nil || m.ctx.Store == nil {
		return nil
	}
	m.isLoading = true
	return []tea.Cmd{func() tea.Msg {
		cats, err := m.ctx.Store.ListCategories(context.Background(), false)
		if err != nil {
			return CategoriesFetchedMsg{SectionID: m.id, Err: err}
		}
		repos, err := m.ctx.Store.ListRepositories(context.Background())
		if err != nil {
			return CategoriesFetchedMsg{SectionID: m.id, Categories: cats, Err: err}
		}
		return CategoriesFetchedMsg{SectionID: m.id, Categories: cats, Repos: repos}
	}}
}

func (m *Model) buildRows(cats []*store.Category, repos []*store.Repository) []section.RowData {
	counts := make(map[string]int)
	for _, r := range repos {
		cat := r.CustomCategory
		if cat == "" {
			cat = r.AICategory
		}
		if cat != "" {
			counts[cat]++
		}
	}
	rows := make([]section.RowData, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, CategoryRow{Category: c, RepoCount: counts[c.ID]})
	}
	return rows
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	switch typed := msg.(type) {
	case CategoriesFetchedMsg:
		if typed.SectionID != m.id {
			return m, nil
		}
		m.categories = typed.Categories
		m.rows = m.buildRows(typed.Categories, typed.Repos)
		m.list.SetRows(m.rows)
		m.loaded = true
		m.isLoading = false
	}
	return m, nil
}

func (m *Model) View() string {
	if !m.loaded {
		return ""
	}
	return m.list.View()
}

func (m *Model) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

func (m *Model) FilterRows(query string) {
	if query == "" {
		m.rows = m.buildRows(m.categories, nil)
		m.list.SetRows(m.rows)
		return
	}
	filtered := make([]section.RowData, 0)
	for _, c := range m.categories {
		if containsFold(c.ID, query) || containsFold(c.Name, query) {
			filtered = append(filtered, CategoryRow{Category: c, RepoCount: m.repoCountForCategory(c.ID)})
		}
	}
	m.rows = filtered
	m.list.SetRows(m.rows)
}

func (m *Model) repoCountForCategory(catID string) int {
	count := 0
	for _, r := range m.rows {
		cr, ok := r.(CategoryRow)
		if ok && cr.Category.ID == catID {
			return cr.RepoCount
		}
	}
	return count
}

func joinKeywords(kws []string) string {
	s := ""
	for i, kw := range kws {
		if i > 0 {
			s += ", "
		}
		if len(s)+len(kw) > 22 {
			s += "..."
			break
		}
		s += kw
	}
	return s
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func containsFold(s, substr string) bool {
	return len(s) >= len(substr) && len(substr) > 0 &&
		lipglossContains(s, substr)
}

func lipglossContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			sc, ss := s[i+j], substr[j]
			if sc >= 'A' && sc <= 'Z' {
				sc += 32
			}
			if ss >= 'A' && ss <= 'Z' {
				ss += 32
			}
			if sc != ss {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
