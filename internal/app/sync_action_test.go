package app

import (
	"context"
	"testing"
	"time"

	"github.com/morehao/starman/internal/store"
)

type stubStore struct {
	repos      []*store.Repository
	syncStats  *store.SyncStats
	syncStates map[string]string
}

func (s *stubStore) UpsertRepository(ctx context.Context, r *store.Repository) error            { return nil }
func (s *stubStore) UpsertRepositories(ctx context.Context, rs []*store.Repository) error        { return nil }
func (s *stubStore) UpsertReposOnSync(ctx context.Context, rs []*store.Repository, fullSync bool) error {
	s.repos = rs
	return nil
}
func (s *stubStore) GetRepository(ctx context.Context, fullName string) (*store.Repository, error) {
	return nil, nil
}
func (s *stubStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	return s.repos, nil
}
func (s *stubStore) ListUnanalyzed(ctx context.Context, limit int) ([]*store.Repository, error)      { return nil, nil }
func (s *stubStore) ListByCategory(ctx context.Context, category string) ([]*store.Repository, error) { return nil, nil }
func (s *stubStore) UpdateAIResult(ctx context.Context, repoID int64, res *store.AIResult) error     { return nil }
func (s *stubStore) UpdateCustomFields(ctx context.Context, repoID int64, f *store.CustomFields) error {
	return nil
}
func (s *stubStore) SetAnalysisFailed(ctx context.Context, repoID int64, failed bool) error { return nil }
func (s *stubStore) DeleteAllRepositories(ctx context.Context) error                         { return nil }
func (s *stubStore) SearchFTS(ctx context.Context, query string, filters *store.SearchFilters) ([]*store.FTSResult, error) {
	return nil, nil
}
func (s *stubStore) RebuildFTSIndex(ctx context.Context) error { return nil }
func (s *stubStore) InsertVector(ctx context.Context, repoID int64, embedding []float64) error {
	return nil
}
func (s *stubStore) SearchVectors(ctx context.Context, queryVec []float64, topK int, threshold float64) ([]store.VectorMatch, error) {
	return nil, nil
}
func (s *stubStore) DeleteVector(ctx context.Context, repoID int64) error { return nil }
func (s *stubStore) SetVectorIndexedAt(ctx context.Context, repoID int64, t time.Time) error {
	return nil
}
func (s *stubStore) GetRepositoryByID(ctx context.Context, id int64) (*store.Repository, error) {
	return nil, nil
}
func (s *stubStore) ListVectorUnindexed(ctx context.Context, limit int) ([]*store.Repository, error) {
	return nil, nil
}
func (s *stubStore) EnsureVec0Dimension(ctx context.Context, dim int) error { return nil }
func (s *stubStore) UpsertRelease(ctx context.Context, r *store.Release) error {
	return nil
}
func (s *stubStore) ListUnreadReleases(ctx context.Context) ([]*store.Release, error)  { return nil, nil }
func (s *stubStore) ListAllReleases(ctx context.Context) ([]*store.Release, error)      { return nil, nil }
func (s *stubStore) ListReleasesByRepo(ctx context.Context, repoFullName string) ([]*store.Release, error) {
	return nil, nil
}
func (s *stubStore) MarkReleaseRead(ctx context.Context, releaseID int64) error { return nil }
func (s *stubStore) MarkAllReleasesRead(ctx context.Context) error {
	return nil
}
func (s *stubStore) SetReleaseSubscription(ctx context.Context, repoFullName string, subscribed bool) error {
	return nil
}
func (s *stubStore) UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error {
	return nil
}
func (s *stubStore) ListCategories(ctx context.Context, visibleOnly bool) ([]*store.Category, error) {
	return nil, nil
}
func (s *stubStore) UpsertCategory(ctx context.Context, c *store.Category) error { return nil }
func (s *stubStore) DeleteCategory(ctx context.Context, id string) error         { return nil }
func (s *stubStore) GetSyncState(ctx context.Context, key string) (string, error) {
	if s.syncStates == nil {
		return "", store.ErrSyncStateNotFound
	}
	v, ok := s.syncStates[key]
	if !ok {
		return "", store.ErrSyncStateNotFound
	}
	return v, nil
}
func (s *stubStore) SetSyncState(ctx context.Context, key, value string) error {
	if s.syncStates == nil {
		s.syncStates = make(map[string]string)
	}
	s.syncStates[key] = value
	return nil
}
func (s *stubStore) SaveSyncStats(ctx context.Context, stats *store.SyncStats) error {
	s.syncStats = stats
	return nil
}
func (s *stubStore) GetSyncStats(ctx context.Context) (*store.SyncStats, error) {
	if s.syncStats == nil {
		return &store.SyncStats{}, nil
	}
	return s.syncStats, nil
}
func (s *stubStore) IncrementSyncCount(ctx context.Context) error {
	if s.syncStats == nil {
		s.syncStats = &store.SyncStats{}
	}
	s.syncStats.TotalSyncCount++
	return nil
}
func (s *stubStore) Close() error { return nil }

type stubStarLister struct {
	listCalls int
	repos     []*store.Repository
}

func (s *stubStarLister) ListStarred(ctx context.Context, username string) ([]*store.Repository, error) {
	s.listCalls++
	return s.repos, nil
}

func TestSyncActionRun(t *testing.T) {
	fakeStore := &stubStore{}
	fakeGitHub := &stubStarLister{
		repos: []*store.Repository{
			{ID: 1, FullName: "owner/repo1"},
			{ID: 2, FullName: "owner/repo2"},
		},
	}
	a := NewSyncAction(fakeStore, fakeGitHub, "testuser")
	res, err := a.Run(context.Background(), SyncOpts{Full: false})
	if err != nil {
		t.Fatal(err)
	}
	if res.Fetched <= 0 {
		t.Fatalf("expected fetched > 0")
	}
	if fakeGitHub.listCalls != 1 {
		t.Fatalf("expected ListStarred called once, got %d", fakeGitHub.listCalls)
	}
}
