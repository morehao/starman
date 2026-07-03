package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/styles"
)

func TestStatusBarShowsCommandPanelShortcuts(t *testing.T) {
	m := NewStatusBar(styles.DefaultTheme())
	_, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	v := m.View()
	for _, token := range []string{"Ctrl+K", "Tab", "Ctrl+Enter", "命令面板", "切换面板"} {
		if !strings.Contains(v, token) {
			t.Fatalf("missing shortcut token %s", token)
		}
	}
}
