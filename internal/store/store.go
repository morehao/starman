package store

import (
	"context"
	"time"
)

type Store interface {
	Close() error

	UpsertRepository(ctx context.Context, r *Repository) error
	UpsertRepositories(ctx context.Context, rs []*Repository) error
	UpsertReposOnSync(ctx context.Context, rs []*Repository, fullSync bool) error
	GetRepository(ctx context.Context, fullName string) (*Repository, error)
	ListRepositories(ctx context.Context) ([]*Repository, error)
	ListUnanalyzed(ctx context.Context, limit int) ([]*Repository, error)
	ListByCategory(ctx context.Context, category string) ([]*Repository, error)
	UpdateAIResult(ctx context.Context, repoID int64, res *AIResult) error
	UpdateCustomFields(ctx context.Context, repoID int64, f *CustomFields) error
	SetAnalysisFailed(ctx context.Context, repoID int64, failed bool) error
	DeleteAllRepositories(ctx context.Context) error

	// Deprecated: SearchFTS used FTS5 for full-text search.
	// Search is now handled by in-memory matching in the ai package (SearchIndex).
	// This method remains for backward compatibility and may be removed in a future version.
	SearchFTS(ctx context.Context, query string, filters *SearchFilters) ([]*FTSResult, error)
	// Deprecated: RebuildFTSIndex was used to refresh the FTS5 full-text index.
	// This method remains for backward compatibility and may be removed in a future version.
	RebuildFTSIndex(ctx context.Context) error

	InsertVector(ctx context.Context, repoID int64, embedding []float64) error
	SearchVectors(ctx context.Context, queryVec []float64, topK int, threshold float64) ([]VectorMatch, error)
	DeleteVector(ctx context.Context, repoID int64) error
	SetVectorIndexedAt(ctx context.Context, repoID int64, t time.Time) error
	GetRepositoryByID(ctx context.Context, id int64) (*Repository, error)
	ListVectorUnindexed(ctx context.Context, limit int) ([]*Repository, error)
	EnsureVec0Dimension(ctx context.Context, dim int) error

	UpsertRelease(ctx context.Context, r *Release) error
	ListUnreadReleases(ctx context.Context) ([]*Release, error)
	ListAllReleases(ctx context.Context) ([]*Release, error)
	ListReleasesByRepo(ctx context.Context, repoFullName string) ([]*Release, error)
	MarkReleaseRead(ctx context.Context, releaseID int64) error
	MarkAllReleasesRead(ctx context.Context) error
	SetReleaseSubscription(ctx context.Context, repoFullName string, subscribed bool) error
	UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error

	ListCategories(ctx context.Context, visibleOnly bool) ([]*Category, error)
	UpsertCategory(ctx context.Context, c *Category) error
	DeleteCategory(ctx context.Context, id string) (int, error)

	GetSyncState(ctx context.Context, key string) (string, error)
	SetSyncState(ctx context.Context, key, value string) error
	SaveSyncStats(ctx context.Context, stats *SyncStats) error
	GetSyncStats(ctx context.Context) (*SyncStats, error)
	IncrementSyncCount(ctx context.Context) error
}
