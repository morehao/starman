package pages

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/generate"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

type genMode int

const (
	genByLanguage genMode = iota
	genByCategory
	genFlat
)

var genModeNames = []string{"By Language", "By Category", "Flat"}

type GenerateModel struct {
	store     store.Store
	theme     *styles.Theme
	generator *generate.Generator
	username  string
	enqueue   func(string) string
	mode      genMode
	preview   string
	width     int
	height    int
	generated bool
	running   bool
}

type generateDoneMsg struct {
	id      string
	preview string
	err     error
}

func NewGenerate(s store.Store, theme *styles.Theme, generator *generate.Generator, username string, enqueue func(string) string) *GenerateModel {
	return &GenerateModel{store: s, theme: theme, generator: generator, username: username, enqueue: enqueue}
}

func (m *GenerateModel) Init() tea.Cmd { return nil }

func (m *GenerateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			m.mode = (m.mode + 1) % 3
		case "enter":
			if m.running {
				return m, nil
			}
			if m.generator == nil {
				m.generated = true
				m.preview = "Error: generate not configured"
				return m, nil
			}
			m.running = true
			id := m.enqueue("generate")
			return m, tea.Batch(
				func() tea.Msg { return types.TaskStartedMsg{ID: id, Label: "Generate"} },
				func() tea.Msg {
					ctx := context.Background()
					var sort generate.SortMode
					switch m.mode {
					case genByCategory:
						sort = generate.SortCategory
					case genFlat:
						sort = generate.SortFlat
					default:
						sort = generate.SortLanguage
					}
					data, err := m.generator.Generate(ctx, generate.Options{
						Username: m.username,
						Sort:     sort,
					})
					if err != nil {
						return generateDoneMsg{id: id, err: err}
					}
					return generateDoneMsg{id: id, preview: string(data)}
				},
			)
		}
	case generateDoneMsg:
		m.running = false
		m.generated = true
		if msg.err != nil {
			m.preview = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.preview = msg.preview
		}
		return m, func() tea.Msg { return types.TaskDoneMsg{ID: msg.id, Err: msg.err} }
	}
	return m, nil
}

func (m *GenerateModel) View() string {
	title := m.theme.PageTitle.Render("Generate") + "\n"
	modes := ""
	for i, name := range genModeNames {
		if genMode(i) == m.mode {
			modes += m.theme.SidebarActive.Render("["+name+"]") + " "
		} else {
			modes += "[" + name + "] "
		}
	}
	body := title + modes + "\n\n"
	if m.generated {
		body += m.preview
	} else {
		body += m.theme.HelpText.Render("Tab: switch mode  Enter: generate  Esc: back")
	}
	return body
}
