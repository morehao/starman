package tui

import (
	"path/filepath"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/morehao/starman/internal/config"
	"github.com/morehao/starman/internal/store"
)

type TuiModel struct {
	config      *config.Config
	store       store.Store
	width       int
	height      int
	currentPage PageID
	lastPage    PageID
	ready       bool
}

func NewTuiModel(cfg *config.Config, s store.Store) *TuiModel {
	return &TuiModel{
		config:      cfg,
		store:       s,
		currentPage: PageDashboard,
		ready:       false,
	}
}

func (m *TuiModel) Init() tea.Cmd {
	return nil
}

func (m *TuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *TuiModel) View() string {
	if !m.ready {
		return "loading..."
	}
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		"starman tui — ready",
	)
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
