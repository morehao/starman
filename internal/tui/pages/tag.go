package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

type TagModel struct {
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

func NewTag(s store.Store, theme *styles.Theme) *TagModel {
	ti := textinput.New()
	ti.Placeholder = "+tag1,-tag2 to edit"
	ti.Width = 40
	return &TagModel{store: s, theme: theme, input: ti, selected: map[int]bool{}}
}

func (m *TagModel) Init() tea.Cmd { return m.loadCmd }

func (m *TagModel) loadCmd() tea.Msg {
	ctx := context.Background()
	repos, _ := m.store.ListRepositories(ctx)
	return reposLoadedMsg{repos: repos}
}

func (m *TagModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				m.applyTags()
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
		}
	}
	return m, nil
}

func (m *TagModel) applyTags() {
	expr := m.input.Value()
	m.input.SetValue("")
	parts := strings.FieldsFunc(expr, func(r rune) bool { return r == ',' })
	var add, del []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "-") {
			del = append(del, p[1:])
		} else if strings.HasPrefix(p, "+") {
			add = append(add, p[1:])
		} else if p != "" {
			add = append(add, p)
		}
	}
	count := 0
	for i, r := range m.repos {
		if m.selected[i] {
			tagSet := map[string]bool{}
			for _, t := range r.CustomTags {
				if t != "" {
					tagSet[t] = true
				}
			}
			for _, a := range add {
				tagSet[a] = true
			}
			for _, d := range del {
				delete(tagSet, d)
			}
			newTags := make([]string, 0, len(tagSet))
			for t := range tagSet {
				newTags = append(newTags, t)
			}
			ctx := context.Background()
			_ = m.store.UpdateCustomFields(ctx, r.ID, &store.CustomFields{
				Description: r.CustomDescription, Tags: newTags,
			})
			m.repos[i].CustomTags = newTags
			count++
		}
	}
	m.status = fmt.Sprintf("Updated %d repos", count)
	m.selected = map[int]bool{}
}

func tagsJoin(tags []string) string {
	return strings.Join(tags, ",")
}

func (m *TagModel) View() string {
	if !m.loaded {
		return "loading repos..."
	}
	title := m.theme.PageTitle.Render("Tag Management") + "\n"
	if m.editing {
		return title + m.input.View() + " (Enter to confirm)\n"
	}
	title += m.theme.HelpText.Render("Space: select  e: edit tags  Esc: back") + "\n"
	if m.status != "" {
		title += m.status + "\n"
	}
	title += "\n"
	start := max(m.cursor-m.height+6, 0)
	end := min(start+m.height-6, len(m.repos))
	for i := start; i < end; i++ {
		r := m.repos[i]
		check := " "
		if m.selected[i] {
			check = "\u25cf"
		}
		line := fmt.Sprintf("%s %-38s %s", check, truncate(r.FullName, 36), tagsJoin(r.CustomTags))
		if i == m.cursor {
			title += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			title += line + "\n"
		}
	}
	return title
}
