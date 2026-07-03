package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/styles"
	"github.com/morehao/starman/internal/tui/types"
)

func TestShortcutPage(t *testing.T) {
	tests := []struct {
		key    string
		want   types.PageID
		wantOK bool
	}{
		{"1", types.PageDashboard, true},
		{"r", types.PageRepoList, true},
		{"t", types.PageTrending, true},
		{"s", types.PageSync, true},
		{"a", types.PageAnalyze, true},
		{"g", types.PageTag, true},
		{"c", types.PageCategorize, true},
		{"S", types.PageStats, true},
		{"R", types.PageRelease, true},
		{"G", types.PageGenerate, true},
		{"b", types.PageBackup, true},
		{"C", types.PageConfig, true},
		{"/", types.PageSearch, true},
		{"x", 0, false},
		{"?", 0, false},
		{"q", 0, false},
		{"", 0, false},
		{"up", 0, false},
		{"enter", 0, false},
	}

	for _, tt := range tests {
		got, ok := ShortcutPage(tt.key)
		if ok != tt.wantOK {
			t.Errorf("ShortcutPage(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
		}
		if ok && got != tt.want {
			t.Errorf("ShortcutPage(%q) page = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestSidebarGroups(t *testing.T) {
	m := NewSidebar(styles.DefaultTheme())
	v := m.View()
	for _, token := range []string{"PINNED", "Dashboard", "Config", "Sync", "Search"} {
		if !strings.Contains(v, token) {
			t.Fatalf("missing %s", token)
		}
	}
}

func TestSidebarRendersRecentAndPinned(t *testing.T) {
	m := NewSidebar(styles.DefaultTheme())
	m.SetRecent([]string{"sync", "analyze"})
	v := m.View()
	for _, token := range []string{"RECENT", "PINNED", "Sync", "Dashboard"} {
		if !strings.Contains(v, token) {
			t.Fatalf("missing %s", token)
		}
	}
}

func TestSidebarShortcutNavigate(t *testing.T) {
	m := NewSidebar(nil)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("sidebar should not return cmd for shortcut keys")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}}
	_, cmd = m.Update(msg)
	if cmd != nil {
		t.Error("sidebar should not return cmd for shortcut keys")
	}
}
