package pages

import (
	"fmt"
	"reflect"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/tui/styles"
)

type ConfigModel struct {
	cfg    *config.Config
	theme  *styles.Theme
	width  int
	height int
}

func NewConfig(cfg *config.Config, theme *styles.Theme) *ConfigModel {
	return &ConfigModel{cfg: cfg, theme: theme}
}

func (m *ConfigModel) Init() tea.Cmd { return nil }

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	}
	return m, nil
}

func mask(s string) string {
	if len(s) > 4 {
		return s[:2] + "***" + s[len(s)-2:]
	}
	return "***"
}

func (m *ConfigModel) View() string {
	title := m.theme.PageTitle.Render("Config") + "\n\n"
	body := ""
	v := reflect.ValueOf(*m.cfg)
	t := reflect.TypeOf(*m.cfg)
	sensitive := map[string]bool{"GitHubToken": true, "AIKey": true, "EmbeddingKey": true}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		val := fmt.Sprintf("%v", v.Field(i).Interface())
		if sensitive[field.Name] {
			val = mask(val)
		}
		body += fmt.Sprintf("  %-25s %s\n", field.Name+":", val)
	}
	return title + body
}
