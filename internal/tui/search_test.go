package tui

import (
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestFilterReposByName(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "cli/go-github", Description: "Go GitHub client"},
		{FullName: "morehao/starman", Description: "Star manager"},
		{FullName: "test/rust-lib", Description: "HTTP server"},
	}
	filtered := filterRepos(repos, "github")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
	if filtered[0].FullName != "cli/go-github" {
		t.Fatalf("unexpected result: %s", filtered[0].FullName)
	}
}

func TestFilterReposByDescription(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", Description: "Fast HTTP server"},
		{FullName: "c/d", Description: "CLI tool"},
	}
	filtered := filterRepos(repos, "http")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
}

func TestFilterReposByLanguage(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/foo", Language: "Go"},
		{FullName: "b/bar", Language: "Rust"},
		{FullName: "c/baz", Language: "Python"},
	}
	filtered := filterRepos(repos, "go")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
}

func TestFilterReposByCategory(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", AICategory: "开发工具"},
		{FullName: "c/d", CustomCategory: "工具"},
	}
	filtered := filterRepos(repos, "开发")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
}

func TestFilterReposByTag(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", AITags: []string{"cli", "tui"}},
		{FullName: "c/d", CustomTags: []string{"web"}},
	}
	filtered := filterRepos(repos, "tui")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
}

func TestFilterReposByTopic(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", Topics: []string{"bubbletea", "golang"}},
		{FullName: "c/d", Topics: []string{"rust", "cli"}},
	}
	filtered := filterRepos(repos, "bubbletea")
	if len(filtered) != 1 {
		t.Fatalf("expected 1 result, got %d", len(filtered))
	}
}

func TestFilterReposCaseInsensitive(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "Owner/Repo"},
	}
	filtered := filterRepos(repos, "owner")
	if len(filtered) != 1 {
		t.Fatal("case-insensitive match failed")
	}
}

func TestFilterReposEmptyQuery(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b"},
		{FullName: "c/d"},
	}
	filtered := filterRepos(repos, "")
	if len(filtered) != 2 {
		t.Fatalf("empty query should return all, got %d", len(filtered))
	}
}

func TestFilterReposNoMatch(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", Description: "tool", Language: "Go"},
	}
	filtered := filterRepos(repos, "zzzzzzz")
	if len(filtered) != 0 {
		t.Fatal("should be no match")
	}
}

func TestMatchRepoFullName(t *testing.T) {
	r := &store.Repository{FullName: "morehao/starman"}
	if !matchRepo(r, "starman") {
		t.Fatal("should match full name")
	}
	if matchRepo(r, "unknown") {
		t.Fatal("should not match")
	}
}

func TestMatchRepoDescription(t *testing.T) {
	r := &store.Repository{Description: "A star manager for GitHub"}
	if !matchRepo(r, "star") {
		t.Fatal("should match description")
	}
}

func TestMatchRepoAITags(t *testing.T) {
	r := &store.Repository{AITags: []string{"tui", "framework"}}
	if !matchRepo(r, "framework") {
		t.Fatal("should match AI tag")
	}
	if matchRepo(r, "nonexistent") {
		t.Fatal("should not match")
	}
}
