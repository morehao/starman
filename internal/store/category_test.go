package store

import (
	"context"
	"testing"
)

func TestListDefaultCategories(t *testing.T) {
	s := testStore(t)
	cats, err := s.ListCategories(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 14 {
		t.Fatalf("expected 14 default categories, got %d", len(cats))
	}
}

func TestUpsertAndDeleteCustomCategory(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := &Category{ID: "my-cat", Name: "My Category", Keywords: []string{"foo", "bar"}, SortOrder: 50, IsCustom: true}
	if err := s.UpsertCategory(ctx, c); err != nil {
		t.Fatal(err)
	}
	cats, _ := s.ListCategories(ctx, false)
	found := false
	for _, cat := range cats {
		if cat.ID == "my-cat" {
			found = true
			if cat.Name != "My Category" || len(cat.Keywords) != 2 {
				t.Fatalf("unexpected category data: %+v", cat)
			}
		}
	}
	if !found {
		t.Fatal("custom category not found")
	}
	if _, err := s.DeleteCategory(ctx, "my-cat"); err != nil {
		t.Fatal(err)
	}
	cats, _ = s.ListCategories(ctx, false)
	for _, cat := range cats {
		if cat.ID == "my-cat" {
			t.Fatal("custom category should be deleted")
		}
	}
}

func TestCannotDeleteDefaultCategory(t *testing.T) {
	s := testStore(t)
	_, err := s.DeleteCategory(context.Background(), "web-app")
	if err == nil {
		t.Fatal("expected error deleting default category")
	}
}

func TestSyncState(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, err := s.GetSyncState(ctx, "last_sync")
	if err != ErrSyncStateNotFound {
		t.Fatalf("expected ErrSyncStateNotFound, got %v", err)
	}
	if err := s.SetSyncState(ctx, "last_sync", "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	val, err := s.GetSyncState(ctx, "last_sync")
	if err != nil {
		t.Fatal(err)
	}
	if val != "2026-01-01T00:00:00Z" {
		t.Fatalf("unexpected value: %s", val)
	}
	if err := s.SetSyncState(ctx, "last_sync", "2026-02-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	val, _ = s.GetSyncState(ctx, "last_sync")
	if val != "2026-02-01T00:00:00Z" {
		t.Fatalf("expected updated value, got %s", val)
	}
}
