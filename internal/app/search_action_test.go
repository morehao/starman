package app

import (
	"context"
	"testing"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/store"
)

type stubSearchService struct{}

func (s *stubSearchService) Search(ctx context.Context, query string, st store.Store, opts ai.SearchOpts) (*ai.SearchResult, error) {
	return &ai.SearchResult{
		Hits: []*ai.SearchHit{
			{Repo: &store.Repository{ID: 1, FullName: "owner/repo1", StargazersCount: 100}, Score: 0.9},
			{Repo: &store.Repository{ID: 2, FullName: "owner/repo2", StargazersCount: 50}, Score: 0.5},
		},
		Mode: ai.SearchModeBasicText,
	}, nil
}

func TestSearchActionRun(t *testing.T) {
	fakeStore := &stubStore{
		repos: []*store.Repository{
			{ID: 1, FullName: "owner/repo1", StargazersCount: 100},
			{ID: 2, FullName: "owner/repo2", StargazersCount: 50},
		},
	}
	fakeAI := &stubSearchService{}
	a := NewSearchAction(fakeStore, fakeAI)
	res, err := a.Run(context.Background(), "cli tool", SearchOpts{Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) == 0 {
		t.Fatalf("expected at least one hit")
	}
}
