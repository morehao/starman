package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/styles"
)

func TestCommandPaletteOpenAndFilter(t *testing.T) {
	p := NewCommandPalette(styles.DefaultTheme(), DefaultCommandCatalog())
	p.Open()
	if !p.IsOpen() {
		t.Fatal("palette should be open")
	}
	p, _ = p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	_, ok := p.Selected()
	if !ok {
		t.Fatal("expected selected command after filtering")
	}
}

func TestCommandPaletteViewIncludesHeader(t *testing.T) {
	p := NewCommandPalette(styles.DefaultTheme(), DefaultCommandCatalog())
	p.Open()
	v := p.View(120, 40)
	for _, token := range []string{"Command Palette", "found"} {
		if !strings.Contains(v, token) {
			t.Fatalf("missing token %s", token)
		}
	}
}
