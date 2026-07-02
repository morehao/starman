package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type CategorizeModel struct {
	store    store.Store
	theme    *styles.Theme
	repos    []*store.Repository
	selected map[int]bool
	cursor   int
	input    textinput.Model
	width    int
	height   int
	loaded   bool
	editing  bool
	status   string
}

func NewCategorize(s store.Store, theme *styles.Theme) *CategorizeModel {
	ti := textinput.New()
	ti.Placeholder = "Enter category name"
	ti.Width = 40
	return &CategorizeModel{store: s, theme: theme, input: ti, selected: map[int]bool{}}
}

func (m *CategorizeModel) Init() tea.Cmd { return m.loadCmd }

func (m *CategorizeModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, _ := m.store.ListRepositories(ctx)
	return reposLoadedMsg{repos: repos}
}

func (m *CategorizeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case reposLoadedMsg:
		m.repos = msg.repos
		m.loaded = true
	case tea.KeyMsg:
		if m.editing {
			m.input, cmd = m.input.Update(msg)
			if msg.String() == "enter" {
				m.editing = false
				m.applyCategory()
			}
			return m, cmd
		}
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(m.repos)-1)
		case " ":
			m.selected[m.cursor] = !m.selected[m.cursor]
		case "e":
			m.editing = true
			m.input.Focus()
			return m, textinput.Blink
		case "l":
			if m.cursor < len(m.repos) {
				r := m.repos[m.cursor]
				r.CategoryLocked = !r.CategoryLocked
				ctx := context.Background()
				_ = m.store.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
					Description: r.CustomDescription, Tags: r.CustomTags,
					Category: r.CustomCategory, CategoryLocked: r.CategoryLocked,
				})
				m.repos[m.cursor] = r
			}
		}
	}
	return m, nil
}

func (m *CategorizeModel) applyCategory() {
	cat := m.input.Value()
	m.input.SetValue("")
	count := 0
	for i, r := range m.repos {
		if m.selected[i] {
			ctx := context.Background()
			_ = m.store.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
				Description: r.CustomDescription, Tags: r.CustomTags,
				Category: cat, CategoryLocked: r.CategoryLocked,
			})
			m.repos[i].CustomCategory = cat
			count++
		}
	}
	m.status = fmt.Sprintf("Updated %d repos to category '%s'", count, cat)
	m.selected = map[int]bool{}
}

func (m *CategorizeModel) View() string {
	if !m.loaded {
		return "loading repos..."
	}
	title := m.theme.PageTitle.Render("Categorize") + "\n"
	if m.editing {
		return title + m.input.View() + " (Enter to confirm)\n"
	}
	title += m.theme.HelpText.Render("Space: select  e: set category  l: toggle lock  Esc: back") + "\n"
	if m.status != "" {
		title += m.status + "\n"
	}
	title += "\n"
	start := max(m.cursor-m.height+6, 0)
	end := min(start+m.height-6, len(m.repos))
	for i := start; i < end; i++ {
		r := m.repos[i]
		check := " "
		lock := " "
		if m.selected[i] {
			check = "\u25cf"
		}
		if r.CategoryLocked {
			lock = "\U0001f512"
		}
		line := fmt.Sprintf("%s %s %-35s %s", check, lock, truncate(r.FullName, 33), r.CustomCategory)
		if i == m.cursor {
			title += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			title += line + "\n"
		}
	}
	return title
}
