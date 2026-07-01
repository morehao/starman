package store

import (
	"context"
	"testing"
)

func testStore(t *testing.T) Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenAndSeedCategories(t *testing.T) {
	s := testStore(t)
	cats, err := s.ListCategories(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != len(defaultCategories) {
		t.Fatalf("expected %d categories, got %d", len(defaultCategories), len(cats))
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/test.db"
	s1, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	s1.Close()
	s2, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	cats, err := s2.ListCategories(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != len(defaultCategories) {
		t.Fatalf("expected %d categories on reopen, got %d", len(defaultCategories), len(cats))
	}
}
