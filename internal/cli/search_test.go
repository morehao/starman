package cli

import (
	"testing"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/store"
)

func TestFilterByCLIOptsByLang(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b", Language: "Go", StargazersCount: 10}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d", Language: "Python", StargazersCount: 20}, Score: 8},
		{Repo: &store.Repository{FullName: "e/f", Language: "Go", StargazersCount: 30}, Score: 3},
	}
	opts := searchOpts{Lang: "Go"}
	filtered := filterByCLIOpts(hits, opts)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 Go repos, got %d", len(filtered))
	}
	for _, h := range filtered {
		if h.Repo.Language != "Go" {
			t.Fatalf("expected only Go repos, got %s", h.Repo.Language)
		}
	}
}

func TestFilterByCLIOptsByCategory(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b", AICategory: "dev-tools"}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d", CustomCategory: "dev-tools"}, Score: 8},
		{Repo: &store.Repository{FullName: "e/f", AICategory: "web-app"}, Score: 3},
	}
	opts := searchOpts{Category: "dev-tools"}
	filtered := filterByCLIOpts(hits, opts)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 dev-tools repos, got %d", len(filtered))
	}
}

func TestFilterByCLIOptsByStars(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b", StargazersCount: 10}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d", StargazersCount: 50}, Score: 3},
		{Repo: &store.Repository{FullName: "e/f", StargazersCount: 30}, Score: 8},
	}
	opts := searchOpts{Sort: "stars"}
	sorted := filterByCLIOpts(hits, opts)
	if len(sorted) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(sorted))
	}
	if sorted[0].Repo.StargazersCount != 50 {
		t.Fatalf("expected 50 stars first, got %d", sorted[0].Repo.StargazersCount)
	}
	if sorted[1].Repo.StargazersCount != 30 {
		t.Fatalf("expected 30 stars second, got %d", sorted[1].Repo.StargazersCount)
	}
}

func TestFilterByCLIOptsLimit(t *testing.T) {
	hits := []*ai.SearchHit{
		{Repo: &store.Repository{FullName: "a/b"}, Score: 5},
		{Repo: &store.Repository{FullName: "c/d"}, Score: 3},
		{Repo: &store.Repository{FullName: "e/f"}, Score: 8},
	}
	opts := searchOpts{Limit: 2}
	limited := filterByCLIOpts(hits, opts)
	if len(limited) != 2 {
		t.Fatalf("expected 2 repos after limit, got %d", len(limited))
	}
	if limited[0].Score != 8 {
		t.Fatalf("expected score 8 first, got %f", limited[0].Score)
	}
}
