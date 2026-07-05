package repoview

import (
	"strings"
	"testing"
	"time"

	"github.com/morehao/starman/internal/store"
)

func TestViewEmpty(t *testing.T) {
	m := NewModel()
	out := m.View()
	if out != "" {
		t.Fatalf("expected empty, got: %q", out)
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

func TestReadmeSection(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		AISummary: "Summary text",
	})
	out := m.View()
	if !strings.Contains(out, "Summary text") {
		t.Fatalf("missing summary: %q", out)
	}
}

func TestReadmeSectionEmpty(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{})
	out := m.View()
	if strings.Contains(out, "README") {
		t.Fatalf("expected no README section when AISummary is empty: %q", out)
	}
}

func TestReadmeSectionGlamour(t *testing.T) {
	m := NewModel()
	m.SetWidth(80)
	m.SetRepo(&store.Repository{
		FullName:      "test/repo",
		AISummary:     "# Hello\n\nThis is a **test** README.\n\n- item 1\n- item 2",
	})
	out := m.View()
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected README content in render: %q", out)
	}
	if !strings.Contains(out, "test") {
		t.Fatalf("expected 'test' in glamour rendered output: %q", out)
	}
}

func TestReadmeSectionGlamourNarrow(t *testing.T) {
	m := NewModel()
	m.SetWidth(30)
	m.SetRepo(&store.Repository{
		FullName:      "test/repo",
		AISummary:     "# Hello\n\nSome markdown content",
	})
	out := m.View()
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected README content in narrow render: %q", out)
	}
}

func TestReleasesSection(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:           "owner/repo",
		SubscribedReleases: true,
	})
	out := m.View()
	if !strings.Contains(out, "Subscribed") {
		t.Fatalf("missing subscribed: %q", out)
	}
}

func TestReleasesSectionNotSubscribed(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{})
	out := m.View()
	if !strings.Contains(out, "Not subscribed") {
		t.Fatalf("missing not subscribed: %q", out)
	}
}

func TestOverviewWithUpdatedAndAnalyzed(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:        "owner/repo",
		StargazersCount: 100,
		ForksCount:      5,
		RepoUpdatedAt:   "2026-01-15T12:00:00Z",
		AnalyzedAt:      &now,
	})
	out := m.View()

	if !strings.Contains(out, "Updated") {
		t.Fatalf("missing Updated label: %q", out)
	}
	if !strings.Contains(out, "2026-01-15 12:00") {
		t.Fatalf("missing formatted Updated time: %q", out)
	}
	if !strings.Contains(out, "Analyzed") {
		t.Fatalf("missing Analyzed label: %q", out)
	}
}

func TestOverviewWithLongDescriptionWrapped(t *testing.T) {
	longDesc := "This is a very long repository description that should be wrapped when the sidebar width is limited to prevent text from overflowing beyond the terminal boundaries and becoming unreadable"
	m := NewModel()
	m.SetWidth(40)
	m.SetRepo(&store.Repository{
		FullName:        "owner/repo",
		StargazersCount: 100,
		ForksCount:      5,
		Description:     longDesc,
	})
	out := m.View()

	if !strings.Contains(out, "overflowing") {
		t.Fatalf("missing description content: %q", out)
	}
	// Verify Description section header exists
	if !strings.Contains(out, "Description") {
		t.Fatalf("missing Description header: %q", out)
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		expect string
	}{
		{
			name:   "empty text",
			text:   "",
			width:  40,
			expect: "",
		},
		{
			name:   "zero width",
			text:   "some text",
			width:  0,
			expect: "some text",
		},
		{
			name:   "text fits within width",
			text:   "short text",
			width:  40,
			expect: "short text",
		},
		{
			name:   "text needs wrapping",
			text:   "hello world foo bar baz",
			width:  10,
			expect: "hello\nworld foo\nbar baz",
		},
		{
			name:   "preserves existing newlines",
			text:   "line one\n\nline two is longer text here",
			width:  15,
			expect: "line one\n\nline two is\nlonger text\nhere",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordWrap(tt.text, tt.width)
			if got != tt.expect {
				t.Fatalf("wordWrap(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.expect)
			}
		})
	}
}
