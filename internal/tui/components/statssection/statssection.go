package statssection

import (
	"context"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/section"
	tuicontext "github.com/morehao/starman/internal/tui/context"
)

type StatsFetchedMsg struct {
	SectionID int
	Repos     []*store.Repository
	Err       error
}

type Model struct {
	id        int
	ctx       *tuicontext.ProgramContext
	cfg       section.SectionConfig
	repos     []*store.Repository
	loaded    bool
	isLoading bool
	tab       int
}

var tabLabels = []string{"by Language", "by Category", "by Tag"}

var tabHeaderLabels = []string{"Language", "Category", "Tag"}

func NewModel(id int, ctx *tuicontext.ProgramContext, cfg section.SectionConfig) *Model {
	return &Model{id: id, ctx: ctx, cfg: cfg}
}

func (m *Model) GetId() int                                          { return m.id }
func (m *Model) GetType() string                                     { return "stats" }
func (m *Model) GetConfig() section.SectionConfig                    { return m.cfg }
func (m *Model) NumRows() int                                        { return 0 }
func (m *Model) CurrRowIndex() int                                   { return 0 }
func (m *Model) GetIsLoading() bool                                  { return m.isLoading }
func (m *Model) GetTotalCount() int                                  { return len(m.repos) }
func (m *Model) IsSearchFocused() bool                               { return false }
func (m *Model) ResetFilters()                                       {}
func (m *Model) SetIsLoading(v bool)                                 { m.isLoading = v }
func (m *Model) UpdateProgramContext(ctx *tuicontext.ProgramContext) { m.ctx = ctx }

func (m *Model) SetSize(w, h int) {}

var stubRow struct {
	section.RowData
}

func (m *Model) CurrRow() section.RowData     { return stubRow }
func (m *Model) NextRow() section.RowData     { return nil }
func (m *Model) PrevRow()                     {}
func (m *Model) FirstItem()                   {}
func (m *Model) LastItem()                    {}
func (m *Model) Pager() string                { return "" }

func (m *Model) NextTab() { m.tab = (m.tab + 1) % len(tabLabels) }
func (m *Model) PrevTab() { m.tab = (m.tab + len(tabLabels) - 1) % len(tabLabels) }

func (m *Model) View() string {
	if !m.loaded {
		return ""
	}

	theme := m.ctx.Theme
	boldStyle := lipgloss.NewStyle().Foreground(theme.PrimaryText).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(theme.FaintText)
	valStyle := lipgloss.NewStyle().Foreground(theme.WarningText)

	w := m.ctx.MainContentWidth
	if w < 20 {
		w = 20
	}

	var b strings.Builder

	var dist map[string]int
	var total int
	switch m.tab {
	case 0:
		dist = countByLanguage(m.repos)
		total = len(m.repos)
	case 1:
		dist = countByCategory(m.repos)
		total = len(m.repos)
	case 2:
		dist = countByTag(m.repos)
	}

	b.WriteString(boldStyle.Render(fmt.Sprintf("Stats  ·  %s  ·  %d repos", tabLabels[m.tab], total)))

	maxBarWidth := w - 30
	if maxBarWidth < 10 {
		maxBarWidth = 10
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("%-18s  %-*s  Count", tabHeaderLabels[m.tab], maxBarWidth, "")))
	b.WriteString("\n\n")

	maxCount := 0
	for _, c := range dist {
		if c > maxCount {
			maxCount = c
		}
	}

	type kv struct {
		key   string
		count int
	}
	var sorted []kv
	for k, v := range dist {
		sorted = append(sorted, kv{k, v})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	for _, item := range sorted {
		label := item.key
		if label == "" {
			label = "(unknown)"
		}
		barLen := 0
		if maxCount > 0 {
			barLen = item.count * maxBarWidth / maxCount
		}
		if barLen < 1 && item.count > 0 {
			barLen = 1
		}
		bar := strings.Repeat("█", barLen)
		line := fmt.Sprintf("%-18s %s %s", valStyle.Render(label), bar, dimStyle.Render(fmt.Sprintf("%d", item.count)))
		b.WriteString(line)
		b.WriteString("\n")
	}

	return lipgloss.NewStyle().Width(w).Render(b.String())
}

func (m *Model) Update(msg tea.Msg) (section.Section, tea.Cmd) {
	switch typed := msg.(type) {
	case StatsFetchedMsg:
		if typed.SectionID != m.id {
			return m, nil
		}
		m.repos = typed.Repos
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
		return StatsFetchedMsg{SectionID: m.id, Repos: repos, Err: err}
	}}
}

func (m *Model) ResetRows() {
	m.repos = nil
	m.loaded = false
	m.isLoading = false
}

func countByLanguage(repos []*store.Repository) map[string]int {
	dist := make(map[string]int)
	for _, r := range repos {
		lang := r.Language
		if lang == "" {
			lang = "(unknown)"
		}
		dist[lang]++
	}
	return dist
}

func countByCategory(repos []*store.Repository) map[string]int {
	dist := make(map[string]int)
	for _, r := range repos {
		cat := r.CustomCategory
		if cat == "" {
			cat = r.AICategory
		}
		if cat == "" {
			cat = "Uncategorized"
		}
		dist[cat]++
	}
	return dist
}

func countByTag(repos []*store.Repository) map[string]int {
	dist := make(map[string]int)
	for _, r := range repos {
		seen := make(map[string]bool)
		for _, t := range r.AITags {
			if t != "" && !seen[t] {
				seen[t] = true
				dist[t]++
			}
		}
		for _, t := range r.CustomTags {
			if t != "" && !seen[t] {
				seen[t] = true
				dist[t]++
			}
		}
	}
	return dist
}
