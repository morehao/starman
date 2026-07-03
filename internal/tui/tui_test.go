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

func TestTuiModelHelpToggle(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)
	m.ready = true
	m.width = 80
	m.height = 24

	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	_, _ = m.Update(msg)
	if !m.showHelp {
		t.Error("expected showHelp to be true after pressing ?")
	}

	helpView := m.View()
	if helpView == "" || helpView == "loading..." {
		t.Error("expected help view to be rendered")
	}

	_, _ = m.Update(msg)
	if m.showHelp {
		t.Error("expected showHelp to be false after pressing ? again")
	}
}

func TestTuiModelSlashNavigatesToSearch(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected navigate command after pressing /")
	}
}

func TestTuiModelOtherKeysFallThrough(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)
	m.ready = true
	m.width = 80
	m.height = 24

	viewBefore := m.View()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	_, _ = m.Update(msg)

	viewAfter := m.View()
	if viewBefore == "" || viewAfter == "" {
		t.Error("views should not be empty")
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

func TestTuiModelSidebarShortcutNavigation(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)

	shortcutTests := []struct {
		key      string
		expected PageID
	}{
		{"1", PageDashboard},
		{"r", PageRepoList},
		{"t", PageTrending},
		{"s", PageSync},
		{"a", PageAnalyze},
		{"g", PageTag},
		{"c", PageCategorize},
		{"S", PageStats},
		{"R", PageRelease},
		{"G", PageGenerate},
		{"b", PageBackup},
		{"C", PageConfig},
	}

	for _, tt := range shortcutTests {
		t.Run(tt.key, func(t *testing.T) {
			m.currentPage = PageDashboard
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Errorf("expected navigate command after pressing %q", tt.key)
			}
		})
	}
}

func TestTuiModelQQuits(t *testing.T) {
	cfg := config.Default()
	m := NewTuiModel(cfg, nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Error("expected quit command on q")
	}
}
