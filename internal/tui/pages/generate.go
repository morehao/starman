package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
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
	mode      genMode
	preview   string
	width     int
	height    int
	generated bool
}

func NewGenerate(s store.Store, theme *styles.Theme) *GenerateModel {
	return &GenerateModel{store: s, theme: theme}
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
			m.preview = fmt.Sprintf("# Awesome Stars\n\nGenerated in %s mode.\n\n- repo1\n- repo2\n", genModeNames[m.mode])
			m.generated = true
		}
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
