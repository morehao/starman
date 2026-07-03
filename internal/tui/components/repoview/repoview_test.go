package repoview

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestViewEmpty(t *testing.T) {
	m := NewModel()
	out := m.View()
	if !strings.Contains(out, "Overview") {
		t.Fatalf("missing tab: %q", out)
	}
}

func TestViewWithRepo(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:        "morehao/starman",
		StargazersCount: 128,
		ForksCount:      12,
		Language:        "Go",
		AICategory:      "开发工具",
		StarredAt:       "2025-06-01",
		Topics:          []string{"tui", "cli"},
		AISummary:       "A terminal UI framework",
	})
	out := m.View()

	if !strings.Contains(out, "morehao/starman") {
		t.Fatalf("missing repo name: %q", out)
	}
	if !strings.Contains(out, "128") {
		t.Fatalf("missing stars: %q", out)
	}
	if !strings.Contains(out, "12") {
		t.Fatalf("missing forks: %q", out)
	}
	if !strings.Contains(out, "Go") {
		t.Fatalf("missing language: %q", out)
	}
}

func TestTabSwitching(t *testing.T) {
	m := NewModel()
	if m.ActiveTab() != 0 {
		t.Fatalf("expected tab 0, got %d", m.ActiveTab())
	}
	m.NextTab()
	if m.ActiveTab() != 1 {
		t.Fatalf("expected tab 1, got %d", m.ActiveTab())
	}
	m.NextTab()
	if m.ActiveTab() != 2 {
		t.Fatalf("expected tab 2, got %d", m.ActiveTab())
	}
	m.NextTab()
	if m.ActiveTab() != 0 {
		t.Fatalf("expected tab 0 after wrap, got %d", m.ActiveTab())
	}

	m.PrevTab()
	if m.ActiveTab() != 2 {
		t.Fatalf("expected tab 2 after prev, got %d", m.ActiveTab())
	}
	m.PrevTab()
	if m.ActiveTab() != 1 {
		t.Fatalf("expected tab 1 after prev, got %d", m.ActiveTab())
	}
}

func TestOverviewRendersAllFields(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:        "owner/repo",
		StargazersCount: 100,
		ForksCount:      5,
		Language:        "Rust",
		AICategory:      "库",
		CategoryLocked:  true,
		AIPlatforms:     []string{"cli"},
		StarredAt:       "2025-01-15",
		Topics:          []string{"parser"},
		AITags:          []string{"regex"},
		CustomTags:      []string{"favorite"},
		AISummary:       "Fast regex parser",
	})

	out := m.View()
	checks := []string{
		"owner/repo", "100", "5", "Rust", "库", "cli",
		"2025-01-15", "parser", "regex", "favorite", "Fast regex parser",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output: %q", check, out)
		}
	}
}

func TestReadmeTab(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		AISummary: "Summary text",
	})
	m.NextTab()
	out := m.View()
	if !strings.Contains(out, "Summary text") {
		t.Fatalf("missing summary: %q", out)
	}
}

func TestReadmeTabEmpty(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{})
	m.NextTab()
	out := m.View()
	if !strings.Contains(out, "No README") {
		t.Fatalf("missing empty message: %q", out)
	}
}

func TestReleasesTab(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:           "owner/repo",
		SubscribedReleases: true,
	})
	m.NextTab()
	m.NextTab()
	out := m.View()
	if !strings.Contains(out, "Subscribed") {
		t.Fatalf("missing subscribed: %q", out)
	}
}

func TestReleasesTabNotSubscribed(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{})
	m.NextTab()
	m.NextTab()
	out := m.View()
	if !strings.Contains(out, "Not subscribed") {
		t.Fatalf("missing not subscribed: %q", out)
	}
}
