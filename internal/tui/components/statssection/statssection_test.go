package statssection

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/store"
	"github.com/morehao/starman/internal/tui/components/section"
	"github.com/morehao/starman/internal/tui/context"
	"github.com/morehao/starman/internal/tui/theme"
)

func TestCountByLanguage(t *testing.T) {
	repos := []*store.Repository{
		{Language: "Go"},
		{Language: "Go"},
		{Language: "Rust"},
		{Language: ""},
	}
	dist := countByLanguage(repos)
	if dist["Go"] != 2 {
		t.Fatalf("expected 2 Go, got %d", dist["Go"])
	}
	if dist["Rust"] != 1 {
		t.Fatalf("expected 1 Rust, got %d", dist["Rust"])
	}
	if dist["(unknown)"] != 1 {
		t.Fatalf("expected 1 unknown, got %d", dist["(unknown)"])
	}
}

func TestCountByCategory(t *testing.T) {
	repos := []*store.Repository{
		{CustomCategory: "工具"},
		{AICategory: "库"},
		{CustomCategory: "工具"},
	}
	dist := countByCategory(repos)
	if dist["工具"] != 2 {
		t.Fatalf("expected 2 工具, got %d", dist["工具"])
	}
	if dist["库"] != 1 {
		t.Fatalf("expected 1 库, got %d", dist["库"])
	}
}

func TestCountByCategoryCustomOverridesAI(t *testing.T) {
	repos := []*store.Repository{
		{CustomCategory: "custom", AICategory: "ai"},
	}
	dist := countByCategory(repos)
	if dist["custom"] != 1 {
		t.Fatalf("custom category should override AI, got %v", dist)
	}
}

func TestCountByTag(t *testing.T) {
	repos := []*store.Repository{
		{AITags: []string{"tui", "cli"}},
		{CustomTags: []string{"tui"}},
		{AITags: []string{"cli"}},
	}
	dist := countByTag(repos)
	if dist["tui"] != 2 {
		t.Fatalf("expected 2 tui, got %d", dist["tui"])
	}
	if dist["cli"] != 2 {
		t.Fatalf("expected 2 cli, got %d", dist["cli"])
	}
}

func TestCountByTagDedup(t *testing.T) {
	repos := []*store.Repository{
		{AITags: []string{"tui"}, CustomTags: []string{"tui"}},
	}
	dist := countByTag(repos)
	if dist["tui"] != 1 {
		t.Fatalf("deduplication failed, got %d", dist["tui"])
	}
}

func TestStatsTabSwitching(t *testing.T) {
	ctx := &context.ProgramContext{Theme: theme.DefaultTheme()}
	m := NewModel(1, ctx, section.SectionConfig{Title: "Stats"})
	if m.tab != 0 {
		t.Fatalf("expected tab 0, got %d", m.tab)
	}
	m.NextTab()
	if m.tab != 1 {
		t.Fatalf("expected tab 1, got %d", m.tab)
	}
	m.PrevTab()
	if m.tab != 0 {
		t.Fatalf("expected tab 0, got %d", m.tab)
	}
	m.PrevTab()
	if m.tab != 2 {
		t.Fatalf("expected tab 2 after wrap, got %d", m.tab)
	}
}

func TestStatsViewRenders(t *testing.T) {
	ctx := &context.ProgramContext{
		Theme:            theme.DefaultTheme(),
		MainContentWidth: 80,
	}
	m := NewModel(1, ctx, section.SectionConfig{Title: "Stats"})
	m.repos = []*store.Repository{
		{Language: "Go"},
		{Language: "Go"},
		{Language: "Rust"},
	}
	m.loaded = true

	out := m.View()
	if !strings.Contains(out, "by Language") {
		t.Fatalf("missing tab label: %q", out)
	}
	if !strings.Contains(out, "Go") {
		t.Fatalf("missing language: %q", out)
	}
}

func TestStatsViewUnloaded(t *testing.T) {
	ctx := &context.ProgramContext{Theme: theme.DefaultTheme()}
	m := NewModel(1, ctx, section.SectionConfig{Title: "Stats"})
	out := m.View()
	if out != "" {
		t.Fatalf("expected empty for unloaded, got %q", out)
	}
}
