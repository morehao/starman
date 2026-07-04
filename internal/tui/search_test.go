package tui

import (
	"testing"
)

func TestConvertSearchHitsToRepos(t *testing.T) {
	hits := []jsonHit{
		{Score: 0.95, FullName: "cli/go-github", Language: "Go", Stars: 4200, Category: "dev-tools", Summary: "GitHub client"},
		{Score: 0.82, FullName: "morehao/starman", Language: "Go", Stars: 150, Category: "tool", Summary: "Star manager"},
	}
	repos := convertSearchHitsToRepos(hits)
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "cli/go-github" {
		t.Fatalf("unexpected name: %s", repos[0].FullName)
	}
	if repos[0].StargazersCount != 4200 {
		t.Fatalf("unexpected stars: %d", repos[0].StargazersCount)
	}
	if repos[1].Language != "Go" {
		t.Fatalf("unexpected language: %s", repos[1].Language)
	}
}

func TestConvertSearchHitsToRepos_Empty(t *testing.T) {
	repos := convertSearchHitsToRepos(nil)
	if len(repos) != 0 {
		t.Fatalf("expected empty, got %d", len(repos))
	}
	repos = convertSearchHitsToRepos([]jsonHit{})
	if len(repos) != 0 {
		t.Fatalf("expected empty, got %d", len(repos))
	}
}

func TestConvertSearchHitsToRepos_EmptyFields(t *testing.T) {
	hits := []jsonHit{
		{FullName: "a/b"},
	}
	repos := convertSearchHitsToRepos(hits)
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	r := repos[0]
	if r.FullName != "a/b" {
		t.Fatalf("unexpected name: %s", r.FullName)
	}
	if r.Language != "" {
		t.Fatalf("expected empty language, got %s", r.Language)
	}
	if r.StargazersCount != 0 {
		t.Fatalf("expected 0 stars, got %d", r.StargazersCount)
	}
}

func TestConvertSearchHitsToRepos_IDCheck(t *testing.T) {
	hits := []jsonHit{
		{Score: 0.9, FullName: "owner/repo", Language: "Rust", Stars: 500, Category: "cli", Summary: "A tool"},
	}
	repos := convertSearchHitsToRepos(hits)
	r := repos[0]
	if r.AICategory != "cli" {
		t.Fatalf("expected category cli, got %s", r.AICategory)
	}
	if r.AISummary != "A tool" {
		t.Fatalf("expected summary, got %s", r.AISummary)
	}
	// IDs are different since we create new pointers
	if r.ID != 0 {
		t.Fatalf("expected ID 0 (not set from search hits), got %d", r.ID)
	}
}

func TestConvertSearchHitsToRepos_ScoreNotStored(t *testing.T) {
	// Score is in jsonHit but not in store.Repository
	hits := []jsonHit{
		{Score: 0.99, FullName: "x/y", Language: "Go", Stars: 1000, Category: "web", Summary: "web framework"},
	}
	repos := convertSearchHitsToRepos(hits)
	if len(repos) != 1 {
		t.Fatalf("expected 1, got %d", len(repos))
	}
}
