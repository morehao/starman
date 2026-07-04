package repodetail

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestViewEmpty(t *testing.T) {
	m := NewModel()
	out := m.View()
	if out != "" {
		t.Fatalf("expected empty for nil repo, got: %q", out)
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

	checks := []string{"morehao/starman", "128", "12", "Go", "开发工具", "tui", "cli", "AI Summary", "framework"}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output: %q", check, out)
		}
	}
}

func TestRendersAllFields(t *testing.T) {
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
		Homepage:        "https://example.com",
	})
	out := m.View()

	checks := []string{
		"owner/repo", "100", "5", "Rust", "库", "cli",
		"2025-01-15", "parser", "regex", "favorite",
		"Fast regex", "https://example.com",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output: %q", check, out)
		}
	}
}

func TestAISummaryEmpty(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName: "noai/repo",
	})
	out := m.View()
	if strings.Contains(out, "AI Summary") {
		t.Fatalf("should not render AI Summary when empty: %q", out)
	}
}

func TestAISummaryGlamour(t *testing.T) {
	m := NewModel()
	m.SetWidth(80)
	m.SetRepo(&store.Repository{
		FullName:  "test/repo",
		AISummary: "# Hello\n\nThis is a **test**.\n\n- item 1\n- item 2",
	})
	out := m.View()
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected glamour rendered content: %q", out)
	}
}

func TestAISummaryGlamourNarrow(t *testing.T) {
	m := NewModel()
	m.SetWidth(16)
	m.SetRepo(&store.Repository{
		FullName:  "test/repo",
		AISummary: "# Hello\n\nSome markdown content",
	})
	out := m.View()
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected content in narrow render: %q", out)
	}
}

func TestReleaseSubscribed(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:           "owner/repo",
		SubscribedReleases: true,
	})
	out := m.View()
	if !strings.Contains(out, "Subscribed") {
		t.Fatalf("missing subscription status: %q", out)
	}
}

func TestReleaseNotSubscribed(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{FullName: "owner/repo"})
	out := m.View()
	if !strings.Contains(out, "Not subscribed") {
		t.Fatalf("missing not subscribed: %q", out)
	}
}

func TestNilRepo(t *testing.T) {
	m := NewModel()
	m.SetWidth(80)
	out := m.View()
	if out != "" {
		t.Fatalf("expected empty string for nil repo, got: %q", out)
	}
}
