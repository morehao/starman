package tui

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components"
	"github.com/morehao/starman/internal/tui/styles"
)

type TuiModel struct {
	config      *config.Config
	store       store.Store
	theme       *styles.Theme
	width       int
	height      int
	currentPage PageID
	sidebar     *components.SidebarModel
	statusbar   *components.StatusBarModel
	ready       bool
}

func NewTuiModel(cfg *config.Config, s store.Store) *TuiModel {
	theme := styles.DefaultTheme()
	return &TuiModel{
		config:      cfg,
		store:       s,
		theme:       theme,
		currentPage: PageDashboard,
		sidebar:     components.NewSidebar(theme),
		statusbar:   components.NewStatusBar(theme),
	}
}

func (m *TuiModel) Init() tea.Cmd {
	return tea.Batch(
		m.sidebar.Init(),
		m.statusbar.Init(),
		tickCmd(),
	)
}

func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case NavigatedMsg:
		m.currentPage = msg.Page
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "/":
			return m, func() tea.Msg { return NavigatedMsg{Page: PageSearch} }
		}
	}

	_, cmd = m.sidebar.Update(msg)
	_, _ = m.statusbar.Update(msg)
	return m, cmd
}

func (m *TuiModel) View() string {
	if !m.ready {
		return "loading..."
	}
	sidebar := m.theme.Sidebar.Render(m.sidebar.View())
	content := m.renderContent()
	status := m.statusbar.View()

	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}

func (m *TuiModel) renderContent() string {
	w := m.width - m.theme.Sidebar.GetWidth()
	return lipgloss.NewStyle().
		Width(w).
		Height(m.height - 1).
		Padding(1).
		Render(fmt.Sprintf("Page: %d\n\nPress / to search, q to quit", m.currentPage))
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func RunTUI(cfg *config.Config) error {
	dir, err := config.DefaultDir()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "starman.db")
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	m := NewTuiModel(cfg, s)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
