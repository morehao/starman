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

	UpsertRelease(ctx context.Context, r *Release) error
	ListUnreadReleases(ctx context.Context) ([]*Release, error)
	ListReleasesByRepo(ctx context.Context, repoFullName string) ([]*Release, error)
	MarkReleaseRead(ctx context.Context, releaseID int64) error
	MarkAllReleasesRead(ctx context.Context) error
	SetReleaseSubscription(ctx context.Context, repoFullName string, subscribed bool) error
	UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error

	ListCategories(ctx context.Context, visibleOnly bool) ([]*Category, error)
	UpsertCategory(ctx context.Context, c *Category) error
	DeleteCategory(ctx context.Context, id string) error

	GetSyncState(ctx context.Context, key string) (string, error)
	SetSyncState(ctx context.Context, key, value string) error
}
