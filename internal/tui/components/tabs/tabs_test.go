package tabs

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/theme"
)

func TestTabsViewContainsSectionTitles(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme: theme.DefaultTheme(),
	}
	m := NewModel(ctx)
	m.SetTitles([]string{"Stars", "Trending", "Releases"})
	out := m.View()
	if !strings.Contains(out, "Stars") {
		t.Fatalf("missing Stars title: %q", out)
	}
	if !strings.Contains(out, "Trending") {
		t.Fatalf("missing Trending title: %q", out)
	}
	if !strings.Contains(out, "Releases") {
		t.Fatalf("missing Releases title: %q", out)
	}
}

func TestTabsViewActiveHighlight(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme: theme.DefaultTheme(),
	}
	m := NewModel(ctx)
	m.SetTitles([]string{"Stars", "Trending"})
	m.SetActive(0)
	out := m.View()
	if out == "" {
		t.Fatalf("empty view")
	}
}
