package app

import (
	"context"
	"time"

	"github.com/morehao/starman/internal/ai"
)

type Result struct {
	Summary  string
	Warnings []string
	Metrics  map[string]string
	Err      error
}

type SyncOpts struct {
	Full     bool
	Watch    bool
	Interval time.Duration
}

type SyncResult struct {
	Result
	Fetched  int
	NewCount int
}

type SearchOpts struct {
	Lang     string
	Category string
	Sort     string
	Limit    int
}

type SearchResult struct {
	Result
	Hits []*ai.SearchHit
}

type SyncAction interface {
	Run(ctx context.Context, opts SyncOpts) (*SyncResult, error)
}

type SearchAction interface {
	Run(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error)
}
