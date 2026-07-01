package ai

import (
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestResolveCategoryCustomLocked(t *testing.T) {
	repo := &store.Repository{CategoryLocked: true, CustomCategory: "我的分类"}
	cats := []*store.Category{{Name: "开发工具", Keywords: []string{"cli"}, IsCustom: false}}
	got := ResolveCategory(repo, []string{"cli"}, cats)
	if got != "我的分类" {
		t.Fatalf("expected locked custom category, got %s", got)
	}
}

func TestResolveCategoryCustomMatch(t *testing.T) {
	repo := &store.Repository{}
	cats := []*store.Category{
		{Name: "开发工具", Keywords: []string{"cli"}, IsCustom: false},
		{Name: "我的工具", Keywords: []string{"mytool"}, IsCustom: true},
	}
	got := ResolveCategory(repo, []string{"mytool"}, cats)
	if got != "我的工具" {
		t.Fatalf("expected custom match, got %s", got)
	}
}

func TestResolveCategoryDefaultMatch(t *testing.T) {
	repo := &store.Repository{}
	cats := []*store.Category{
		{Name: "开发工具", Keywords: []string{"cli"}, IsCustom: false},
	}
	got := ResolveCategory(repo, []string{"cli", "tool"}, cats)
	if got != "开发工具" {
		t.Fatalf("expected default match, got %s", got)
	}
}

func TestResolveCategoryNoMatch(t *testing.T) {
	repo := &store.Repository{}
	cats := []*store.Category{
		{Name: "开发工具", Keywords: []string{"cli"}, IsCustom: false},
	}
	got := ResolveCategory(repo, []string{"cooking"}, cats)
	if got != "others" {
		t.Fatalf("expected others, got %s", got)
	}
}

func TestResolveCategoryLockedNoMatchPreservesAICategory(t *testing.T) {
	repo := &store.Repository{CategoryLocked: true, AICategory: "dev-tools"}
	cats := []*store.Category{
		{Name: "开发工具", Keywords: []string{"cli"}, IsCustom: false},
	}
	got := ResolveCategory(repo, []string{"cooking"}, cats)
	if got != "dev-tools" {
		t.Fatalf("expected locked repo to preserve AICategory, got %s", got)
	}
}

func TestResolveCategoryLockedNoMatchFallbackToOthers(t *testing.T) {
	repo := &store.Repository{CategoryLocked: true}
	cats := []*store.Category{
		{Name: "开发工具", Keywords: []string{"cli"}, IsCustom: false},
	}
	got := ResolveCategory(repo, []string{"cooking"}, cats)
	if got != "others" {
		t.Fatalf("expected others when no category set on locked repo, got %s", got)
	}
}
