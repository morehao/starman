package app

import (
	"context"
	"fmt"

	"github.com/morehao/starman/internal/ai"
	"github.com/morehao/starman/internal/store"
)

type SearchService interface {
	Search(ctx context.Context, query string, st store.Store, opts ai.SearchOpts) (*ai.SearchResult, error)
}

type searchAction struct {
	store store.Store
	svc   SearchService
}

func NewSearchAction(s store.Store, svc SearchService) SearchAction {
	return &searchAction{store: s, svc: svc}
}

func (a *searchAction) Run(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) {
	aiOpts := ai.SearchOpts{
		Language: opts.Lang,
		Category: opts.Category,
		Sort:     opts.Sort,
		Limit:    opts.Limit,
	}

	result, err := a.svc.Search(ctx, query, a.store, aiOpts)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &SearchResult{
		Result: Result{
			Summary: fmt.Sprintf("%d results for \"%s\"", len(result.Hits), query),
		},
		Hits: result.Hits,
	}, nil
}
