package backup

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/morehao/starman/internal/store"
)

type mockStore struct {
	repos []*store.Repository
	cats  []*store.Category
	store.Store
}

func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	return m.repos, nil
}

func (m *mockStore) ListUnreadReleases(ctx context.Context) ([]*store.Release, error) {
	return nil, nil
}

func (m *mockStore) ListCategories(ctx context.Context, visibleOnly bool) ([]*store.Category, error) {
	return m.cats, nil
}

func (m *mockStore) UpsertRepository(ctx context.Context, r *store.Repository) error {
	return nil
}

func (m *mockStore) UpsertCategory(ctx context.Context, c *store.Category) error {
	return nil
}

func TestExportJSON(t *testing.T) {
	s := &mockStore{
		repos: []*store.Repository{{ID: 1, FullName: "owner/repo1"}},
		cats:  []*store.Category{{ID: "web-app", Name: "Web 应用"}},
	}
	data, err := ExportJSON(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	if b.Version != 1 {
		t.Fatalf("expected version 1, got %d", b.Version)
	}
	if len(b.Repositories) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(b.Repositories))
	}
	if len(b.Categories) != 1 {
		t.Fatalf("expected 1 category, got %d", len(b.Categories))
	}
}

func TestImportJSONMerge(t *testing.T) {
	s := &mockStore{}
	data := []byte(`{"version":1,"exported_at":"2026-01-01T00:00:00Z","repositories":[{"id":1,"full_name":"owner/repo1"}],"releases":[],"categories":[{"id":"web-app","name":"Web"}]}`)
	if err := ImportJSON(context.Background(), s, data, ImportMerge); err != nil {
		t.Fatal(err)
	}
}
