package ai

import (
	"context"
	"fmt"

	"github.com/morehao/starman/internal/store"
)

func (s *Service) basicTextSearch(
	ctx context.Context, query string, st store.Store, opts SearchOpts,
) (*SearchResult, error) {
	filters := buildSearchFilters(opts, nil)
	filters.Limit = 50

	ftsResults, err := st.SearchFTS(ctx, query, filters)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	hits := make([]*SearchHit, 0, len(ftsResults))
	for _, fr := range ftsResults {
		hits = append(hits, &SearchHit{Repo: fr.Repo, Score: fr.BM25Score})
	}

	sortHits(hits, opts.Sort)
	if opts.Limit > 0 && opts.Limit < len(hits) {
		hits = hits[:opts.Limit]
	}

	return &SearchResult{Hits: hits}, nil
}

func buildSearchFilters(opts SearchOpts, intent *QueryIntent) *store.SearchFilters {
	filters := &store.SearchFilters{
		Language:       opts.Language,
		Category:       opts.Category,
		Platform:       opts.Platform,
		Tags:           opts.Tags,
		MinStars:       opts.MinStars,
		MaxStars:       opts.MaxStars,
		Analyzed:       opts.Analyzed,
		AnalysisFailed: opts.AnalysisFailed,
	}
	if intent != nil {
		if filters.Language == "" {
			filters.Language = intent.Language
		}
		if filters.Category == "" {
			filters.Category = intent.Category
		}
		if filters.Platform == "" {
			filters.Platform = intent.Platform
		}
		if filters.MinStars == 0 {
			filters.MinStars = intent.MinStars
		}
		if filters.MaxStars == 0 {
			filters.MaxStars = intent.MaxStars
		}
	}
	return filters
}
