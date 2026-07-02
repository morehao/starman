package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type searchMode int

const (
	modeText searchMode = iota
	modeVector
	modeLLM
)

var modeNames = []string{"TEXT", "VECTOR", "LLM"}

type SearchModel struct {
	store   store.Store
	theme   *styles.Theme
	input   textinput.Model
	results []*store.Repository
	cursor  int
	mode    searchMode
	width   int
	height  int
	loaded  bool
}

func NewSearch(s store.Store, theme *styles.Theme) *SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search repositories..."
	ti.CharLimit = 100
	ti.Width = 60
	return &SearchModel{store: s, theme: theme, input: ti}
}

func (m *SearchModel) Init() tea.Cmd { return textinput.Blink }

func (m *SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.mode = (m.mode + 1) % 3
		case "esc":
			return m, func() tea.Msg { return types.NavigatedMsg{Page: types.PageDashboard} }
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.results)-1)
		case "enter":
			if m.cursor < len(m.results) {
				return m, func() tea.Msg {
					return types.RepoSelectedMsg{FullName: m.results[m.cursor].FullName}
				}
			}
		}
	}
	m.input, cmd = m.input.Update(msg)
	m.doSearch()
	return m, cmd
}

func (m *SearchModel) doSearch() {
	q := strings.TrimSpace(m.input.Value())
	if len(q) < 2 {
		m.results = nil
		return
	}
	ctx := context.Background()
	repos, err := m.store.ListRepositories(ctx)
	if err != nil {
		return
	}
	var filtered []*store.Repository
	qLower := strings.ToLower(q)
	for _, r := range repos {
		if strings.Contains(strings.ToLower(r.FullName), qLower) ||
			strings.Contains(strings.ToLower(r.Description), qLower) ||
			strings.Contains(strings.ToLower(r.AISummary), qLower) ||
			strings.Contains(strings.ToLower(r.Language), qLower) {
			filtered = append(filtered, r)
		}
	}
	m.results = filtered
	m.loaded = true
	m.cursor = 0
}

func (m *SearchModel) View() string {
	title := m.theme.PageTitle.Render("Search") + "\n"
	modeStr := ""
	for i, name := range modeNames {
		if searchMode(i) == m.mode {
			modeStr += m.theme.SidebarActive.Render("[" + name + "]") + " "
		} else {
			modeStr += "[" + name + "] "
		}
	}
	body := title + m.input.View() + "\n" + modeStr + "\n\n"
	if m.loaded && len(m.results) > 0 {
		body += fmt.Sprintf("Results: %d\n", len(m.results))
		for i, r := range m.results {
			line := fmt.Sprintf("  %-40s %-10s \u2605%d", truncate(r.FullName, 38), r.Language, r.StargazersCount)
			if i == m.cursor {
				body += m.theme.SidebarActive.Render(line) + "\n"
			} else {
				body += line + "\n"
			}
		}
	}
	return body
}
