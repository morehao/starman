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

func TestTabs_StarsSections(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:       theme.DefaultTheme(),
		ScreenWidth: 120,
	}
	sections := []string{"All", "Language", "Category", "Tag"}
	m := NewModel(ctx)
	m.SetTitles([]string{"Stars", "Trending", "Releases", "Stats"})
	m.SetSectionTabs(sections)
	m.SetActiveSection(1)
	view := m.View()
	for _, s := range sections {
		if !strings.Contains(view, s) {
			t.Errorf("expected section tab %q in view", s)
		}
	}
}

func TestTabs_SectionNavigation(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:       theme.DefaultTheme(),
		ScreenWidth: 120,
	}
	sections := []string{"All", "Language", "Category", "Tag"}

	m := NewModel(ctx)
	m.SetSectionTabs(sections)
	if m.ActiveSectionIndex() != 0 {
		t.Fatal("expected active section 0")
	}

	m.NextSection()
	if m.ActiveSectionIndex() != 1 {
		t.Fatalf("expected active section 1, got %d", m.ActiveSectionIndex())
	}

	m.PrevSection()
	if m.ActiveSectionIndex() != 0 {
		t.Fatalf("expected active section 0, got %d", m.ActiveSectionIndex())
	}
}
