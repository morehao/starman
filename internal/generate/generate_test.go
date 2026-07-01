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
		{FullName: "owner/repo1", URL: "https://github.com/owner/repo1", Language: "Go", Description: "A Go tool"},
		{FullName: "owner/repo2", URL: "https://github.com/owner/repo2", Language: "", Description: "No language"},
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
}

func TestGenerateByCategory(t *testing.T) {
	s := &mockStore{repos: []*store.Repository{
		{FullName: "owner/repo1", URL: "https://github.com/owner/repo1", AICategory: "开发工具", AISummary: "好工具", AITags: []string{"cli"}},
	}}
	g := NewGenerator(s)
	out, err := g.Generate(context.Background(), Options{Username: "testuser", Sort: SortCategory})
	if err != nil {
		t.Fatal(err)
	}
	str := string(out)
	if !strings.Contains(str, "## 开发工具") {
		t.Fatal("expected 开发工具 section")
	}
	if !strings.Contains(str, "好工具") {
		t.Fatal("expected AI summary")
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
