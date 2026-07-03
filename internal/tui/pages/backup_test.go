package pages

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/styles"
)

func TestBackupImportRequiresConfirm(t *testing.T) {
	m := NewBackup(nil, styles.DefaultTheme(), nil, nil)
	m.cursor = 1
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("expected confirm step before task execution")
	}
	if !m.confirmed {
		t.Fatalf("expected confirmed flag after first Enter")
	}
	if m.confirmPrompt == "" {
		t.Fatalf("expected confirm prompt")
	}
}
