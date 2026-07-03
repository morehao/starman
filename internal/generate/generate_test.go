package generate

import (
	"context"
	"strings"
	"testing"

	"github.com/morehao/starman/internal/store"
)

type mockStore struct {
	repos []*store.Repository
	store.Store
}

func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	return m.repos, nil
}

func TestGenerateByLanguage(t *testing.T) {
	s := &mockStore{repos: []*store.Repository{
		{
			FullName:        "owner/repo1",
			URL:             "https://github.com/owner/repo1",
			Language:        "Go",
			AISummary:       "A Go tool",
			StargazersCount: 12345,
			Topics:          []string{"cli", "devtools"},
			Homepage:        "https://example.com",
			AIPlatforms:     []string{"cli"},
		},
		{
			FullName: "owner/repo2",
			URL:      "https://github.com/owner/repo2",
			Language: "",
		},
	}}
	g := NewGenerator(s)
	out, err := g.Generate(context.Background(), Options{Username: "testuser", Sort: SortLanguage})
	if err != nil {
		t.Fatal(err)
	}
	str := string(out)
	if !strings.Contains(str, "## Go") {
		t.Fatal("expected ## Go section")
	}
	if !strings.Contains(str, "## Others") {
		t.Fatal("expected ## Others section")
	}
	if !strings.Contains(str, "[owner/repo1]") {
		t.Fatal("expected repo1 link")
	}
	if !strings.Contains(str, "A Go tool") {
		t.Fatal("expected AI summary")
	}
	if !strings.Contains(str, "12.3k") {
		t.Fatal("expected formatted stars")
	}
	if !strings.Contains(str, "[site](https://example.com)") {
		t.Fatal("expected homepage")
	}
	if !strings.Contains(str, "`cli`") {
		t.Fatal("expected tag cli")
	}
	if !strings.Contains(str, "`devtools`") {
		t.Fatal("expected tag devtools")
	}
}

func TestGenerateByCategory(t *testing.T) {
	s := &mockStore{repos: []*store.Repository{
		{
			FullName:        "owner/repo1",
			URL:             "https://github.com/owner/repo1",
			Language:        "TypeScript",
			AISummary:       "好工具",
			AITags:          []string{"frontend"},
			Topics:          []string{"react"},
			CustomTags:      []string{"favorite"},
			AIPlatforms:     []string{"web"},
			StargazersCount: 500,
		},
	}}
	g := NewGenerator(s)
	out, err := g.Generate(context.Background(), Options{Username: "testuser", Sort: SortCategory})
	if err != nil {
		t.Fatal(err)
	}
	str := string(out)
	if !strings.Contains(str, "好工具") {
		t.Fatal("expected AI summary")
	}
	if !strings.Contains(str, "TypeScript") {
		t.Fatal("expected language")
	}
	if !strings.Contains(str, `⭐ 500`) {
		t.Fatal("expected stars without k suffix")
	}
	if !strings.Contains(str, "`react`") {
		t.Fatal("expected GitHub topic react")
	}
	if !strings.Contains(str, "`frontend`") {
		t.Fatal("expected AI tag frontend")
	}
	if !strings.Contains(str, "`favorite`") {
		t.Fatal("expected custom tag favorite")
	}
}

func TestGenerateFlat(t *testing.T) {
	s := &mockStore{repos: []*store.Repository{
		{
			FullName:  "owner/repo1",
			URL:       "https://github.com/owner/repo1",
			Language:  "Rust",
			AISummary: "A Rust tool",
		},
	}}
	g := NewGenerator(s)
	out, err := g.Generate(context.Background(), Options{Username: "testuser", Sort: SortFlat})
	if err != nil {
		t.Fatal(err)
	}
	str := string(out)
	if !strings.Contains(str, "## Repositories") {
		t.Fatal("expected Repositories header")
	}
	if !strings.Contains(str, "A Rust tool") {
		t.Fatal("expected AI summary")
	}
	if !strings.Contains(str, "Rust") {
		t.Fatal("expected language")
	}
}

func TestFormatStars(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{10000, "10.0k"},
		{42300, "42.3k"},
	}
	for _, tt := range tests {
		if got := formatStars(tt.n); got != tt.want {
			t.Errorf("formatStars(%d) = %s, want %s", tt.n, got, tt.want)
		}
	}
}

func TestMergeTags(t *testing.T) {
	got := mergeTags(
		[]string{"go", "cli"},
		[]string{"cli", " devtools "},
		[]string{"favorite"},
	)
	if len(got) != 4 {
		t.Fatalf("expected 4 tags, got %d: %v", len(got), got)
	}
	// 验证去重
	seen := make(map[string]bool)
	for _, tag := range got {
		if seen[tag] {
			t.Errorf("duplicate tag: %s", tag)
		}
		seen[tag] = true
	}
}

func TestCapitalize(t *testing.T) {
	if got := capitalize("javascript"); got != "JavaScript" {
		t.Fatalf("expected JavaScript, got %s", got)
	}
	if got := capitalize("go"); got != "Go" {
		t.Fatalf("expected Go, got %s", got)
	}
}
