package starssection

import (
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestGroupByLanguage(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/go1", Language: "Go"},
		{FullName: "b/go2", Language: "Go"},
		{FullName: "c/rust1", Language: "Rust"},
		{FullName: "d/unknown", Language: ""},
	}
	got := GroupByLanguage(repos)
	if got.GroupCount() != 3 {
		t.Fatalf("expected 3 groups, got %d", got.GroupCount())
	}
	if got.BucketCount("Go") != 2 {
		t.Errorf("Go count = %d, want 2", got.BucketCount("Go"))
	}
	if got.BucketCount("Rust") != 1 {
		t.Errorf("Rust count = %d, want 1", got.BucketCount("Rust"))
	}
	if got.BucketCount("") != 1 {
		t.Errorf("empty language count = %d, want 1", got.BucketCount(""))
	}
}

func TestGroupByCategory(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/repo1", CustomCategory: "tools", AICategory: ""},
		{FullName: "b/repo2", CustomCategory: "", AICategory: "tools"},
		{FullName: "c/repo3", CustomCategory: "", AICategory: ""},
	}
	got := GroupByCategory(repos)
	if got.BucketCount("tools") != 2 {
		t.Errorf("tools count = %d, want 2", got.BucketCount("tools"))
	}
	if got.BucketCount(Uncategorized) != 1 {
		t.Errorf("uncategorized count = %d, want 1", got.BucketCount(Uncategorized))
	}
}

func TestGroupByTag(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/repo1", AITags: []string{"tui", "cli"}},
		{FullName: "b/repo2", CustomTags: []string{"cli", "awesome"}},
		{FullName: "c/repo3"},
	}
	got := GroupByTag(repos)
	if got.BucketCount("tui") != 1 {
		t.Errorf("tui count = %d, want 1", got.BucketCount("tui"))
	}
	if got.BucketCount("cli") != 2 {
		t.Errorf("cli count = %d, want 2", got.BucketCount("cli"))
	}
	if got.BucketCount(Untagged) != 1 {
		t.Errorf("untagged count = %d, want 1", got.BucketCount(Untagged))
	}
}

func TestGroupKeysSorted(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/r1", Language: "Zig"},
		{FullName: "b/r2", Language: "Ada"},
		{FullName: "c/r3", Language: "Go"},
	}
	got := GroupByLanguage(repos)
	keys := got.KeysSorted()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0] != "Ada" || keys[1] != "Go" || keys[2] != "Zig" {
		t.Errorf("keys not sorted: %v", keys)
	}
}
