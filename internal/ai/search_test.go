package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/morehao/starman/internal/store"
)

func TestSearch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(completionResponse{
				Choices: []struct {
					Message Message `json:"message"`
				}{{Message: Message{Content: `{"fts_query":"terminal cli","keywords":["terminal","cli"],"language":"","category":"","platform":"","min_stars":0,"max_stars":0}`}}},
			})
		} else {
			json.NewEncoder(w).Encode(completionResponse{
				Choices: []struct {
					Message Message `json:"message"`
				}{{Message: Message{Content: `{"rankings":[{"index":0,"score":9.5}]}`}}},
			})
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "key", "model")
	svc := NewService(c, nil)

	var st mockStore
	hits, err := svc.Search(context.Background(), "终端工具", &st, SearchOpts{Rerank: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Repo.ID != 1 {
		t.Fatalf("expected repo 1, got %d", hits[0].Repo.ID)
	}
	if hits[0].Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestCalcWeightedScore(t *testing.T) {
	r := &store.Repository{
		FullName:     "owner/awesome-cli",
		Description:  "A CLI tool",
		AISearchText: "一个强大的命令行工具，用于自动化开发流程",
		StargazersCount: 1000,
	}
	score := calcWeightedScore(10.0, r.StargazersCount, r, []string{"cli"})
	if score <= 0 {
		t.Fatalf("expected positive score, got %f", score)
	}
}

type mockStore struct{}

func (m *mockStore) Close() error { return nil }

func (m *mockStore) UpsertRepository(ctx context.Context, r *store.Repository) error { return nil }
func (m *mockStore) UpsertRepositories(ctx context.Context, rs []*store.Repository) error { return nil }
func (m *mockStore) UpsertReposOnSync(ctx context.Context, rs []*store.Repository, fullSync bool) error { return nil }
func (m *mockStore) GetRepository(ctx context.Context, fullName string) (*store.Repository, error) { return nil, nil }
func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) { return nil, nil }
func (m *mockStore) ListUnanalyzed(ctx context.Context, limit int) ([]*store.Repository, error) { return nil, nil }
func (m *mockStore) ListByCategory(ctx context.Context, category string) ([]*store.Repository, error) { return nil, nil }
func (m *mockStore) UpdateAIResult(ctx context.Context, repoID int64, res *store.AIResult) error { return nil }
func (m *mockStore) UpdateCustomFields(ctx context.Context, repoID int64, f *store.CustomFields) error { return nil }
func (m *mockStore) SetAnalysisFailed(ctx context.Context, repoID int64, failed bool) error { return nil }
func (m *mockStore) DeleteAllRepositories(ctx context.Context) error { return nil }
func (m *mockStore) UpsertRelease(ctx context.Context, r *store.Release) error { return nil }
func (m *mockStore) ListUnreadReleases(ctx context.Context) ([]*store.Release, error) { return nil, nil }
func (m *mockStore) ListAllReleases(ctx context.Context) ([]*store.Release, error) { return nil, nil }
func (m *mockStore) ListReleasesByRepo(ctx context.Context, repoFullName string) ([]*store.Release, error) { return nil, nil }
func (m *mockStore) MarkReleaseRead(ctx context.Context, releaseID int64) error { return nil }
func (m *mockStore) MarkAllReleasesRead(ctx context.Context) error { return nil }
func (m *mockStore) SetReleaseSubscription(ctx context.Context, repoFullName string, subscribed bool) error { return nil }
func (m *mockStore) UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error { return nil }
func (m *mockStore) ListCategories(ctx context.Context, visibleOnly bool) ([]*store.Category, error) { return nil, nil }
func (m *mockStore) UpsertCategory(ctx context.Context, c *store.Category) error { return nil }
func (m *mockStore) DeleteCategory(ctx context.Context, id string) error { return nil }
func (m *mockStore) GetSyncState(ctx context.Context, key string) (string, error) { return "", nil }
func (m *mockStore) SetSyncState(ctx context.Context, key, value string) error { return nil }
func (m *mockStore) SaveSyncStats(ctx context.Context, stats *store.SyncStats) error { return nil }
func (m *mockStore) GetSyncStats(ctx context.Context) (*store.SyncStats, error) { return &store.SyncStats{}, nil }
func (m *mockStore) IncrementSyncCount(ctx context.Context) error { return nil }
func (m *mockStore) SearchFTS(ctx context.Context, query string, filters *store.SearchFilters) ([]*store.FTSResult, error) {
	now := time.Now()
	return []*store.FTSResult{
		{Repo: &store.Repository{ID: 1, FullName: "owner/cli-tool", Description: "A terminal tool", AITags: []string{"cli"}, Topics: []string{"go"}, AISearchText: "一个命令行工具", AnalyzedAt: &now}, BM25Score: 8.5},
	}, nil
}
func (m *mockStore) RebuildFTSIndex(ctx context.Context) error { return nil }
