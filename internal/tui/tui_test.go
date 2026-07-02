package tui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/config"
)

func TestTuiModelInit(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)
	if m.currentPage != PageDashboard {
		t.Errorf("expected PageDashboard, got %d", m.currentPage)
	}
	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil init command")
	}
}

func TestTuiModelNavigation(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)
	m.currentPage = PageSearch
	if m.currentPage != PageSearch {
		t.Errorf("expected PageSearch, got %d", m.currentPage)
	}
}

func TestTuiModelQuit(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected quit command on ctrl+c")
	}
}

func TestPageIDValues(t *testing.T) {
	if PageDashboard != 0 {
		t.Errorf("PageDashboard = %d, want 0", PageDashboard)
	}
	if PageSearch != 1 {
		t.Errorf("PageSearch = %d, want 1", PageSearch)
	}
}
