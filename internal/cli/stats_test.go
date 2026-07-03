package cli

import (
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestAggregateStatsByLanguage(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", Language: "Go"},
		{FullName: "c/d", Language: "Go"},
		{FullName: "e/f", Language: "TypeScript"},
		{FullName: "g/h", Language: ""},
	}
	items := aggregateStats(repos, "language")
	if len(items) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(items))
	}
	if items[0].Name != "Go" || items[0].Count != 2 {
		t.Fatalf("expected Go count 2 first, got %+v", items[0])
	}
	if items[1].Name != "Others" || items[1].Count != 1 {
		t.Fatalf("expected Others count 1 second, got %+v", items[1])
	}
}

func TestAggregateStatsByCategory(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", AICategory: "dev-tools"},
		{FullName: "c/d", CustomCategory: "dev-tools"},
		{FullName: "e/f"},
	}
	items := aggregateStats(repos, "category")
	if len(items) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(items))
	}
	if items[0].Count != 2 {
		t.Fatalf("expected count 2 for top category, got %d", items[0].Count)
	}
	if items[1].Name != "其他" || items[1].Count != 1 {
		t.Fatalf("expected 其他 count 1 second, got %+v", items[1])
	}
}

func TestAggregateStatsByTag(t *testing.T) {
	repos := []*store.Repository{
		{FullName: "a/b", AITags: []string{"cli", "go"}, CustomTags: []string{"awesome"}},
		{FullName: "c/d", AITags: []string{"cli"}, CustomTags: []string{}},
		{FullName: "e/f", AITags: []string{}, CustomTags: []string{"web"}},
	}
	items := aggregateStats(repos, "tag")
	if len(items) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(items))
	}
	if items[0].Name != "cli" || items[0].Count != 2 {
		t.Fatalf("expected cli count 2 first, got %+v", items[0])
	}
}
