package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/styles"
)

var backupOps = []string{
	"Export JSON",
	"Import JSON",
	"Push to WebDAV",
	"Pull from WebDAV",
}

type BackupModel struct {
	store  store.Store
	theme  *styles.Theme
	cursor int
	status string
	width  int
	height int
}

func NewBackup(s store.Store, theme *styles.Theme) *BackupModel {
	return &BackupModel{store: s, theme: theme}
}

func (m *BackupModel) Init() tea.Cmd { return nil }

func (m *BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width - m.theme.Sidebar.GetWidth() - 4
		m.height = msg.Height - 3
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.cursor = max(m.cursor-1, 0)
		case "down", "j":
			m.cursor = min(m.cursor+1, len(backupOps)-1)
		case "enter":
			m.status = fmt.Sprintf("Running: %s...", backupOps[m.cursor])
		}
	}
	return m, nil
}

func (m *BackupModel) View() string {
	title := m.theme.PageTitle.Render("Backup") + "\n\n"
	var body string
	for i, op := range backupOps {
		line := fmt.Sprintf("  %s", op)
		if i == m.cursor {
			body += m.theme.SidebarActive.Render(line) + "\n"
		} else {
			body += line + "\n"
		}
	}
	if m.status != "" {
		body += "\n" + m.status
	}
	return title + body
}
