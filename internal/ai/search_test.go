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
				}{{Message: Message{Content: `{"fts_query":"终端 工具 terminal cli","keywords":["终端","工具","terminal","cli"],"language":"","category":"","platform":"cli","min_stars":0,"max_stars":0}`}}},
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
	searchIndex := NewSearchIndex()
	if err := searchIndex.Load(context.Background(), &st); err != nil {
		t.Fatal(err)
	}
	svc.SetSearchIndex(searchIndex)

	result, err := svc.Search(context.Background(), "终端工具", &st, SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	hits := result.Hits
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

func TestScoreRepo(t *testing.T) {
	entry := &repoEntry{
		repo: &store.Repository{
			FullName:        "owner/awesome-cli",
			Description:     "A CLI tool for development automation",
			AISearchText:    "一个强大的命令行工具，用于自动化开发流程",
			StargazersCount: 1000,
		},
		searchText: "owner/awesome-cli a cli tool for development automation 一个强大的命令行工具，用于自动化开发流程 go cli",
	}
	score := scoreRepo(entry, "终端 工具 terminal cli", []string{"cli", "terminal"})
	if score <= 0 {
		t.Fatalf("expected positive score, got %f", score)
	}
}

// Deprecated: TestCalcWeightedScore tests the deprecated FTS5 scoring function.
func TestCalcWeightedScore(t *testing.T) {
	r := &store.Repository{
		FullName:        "owner/awesome-cli",
		Description:     "A CLI tool",
		AISearchText:    "一个强大的命令行工具，用于自动化开发流程",
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
func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	now := time.Now()
	return []*store.Repository{
		{ID: 1, FullName: "owner/cli-tool", Description: "A terminal tool", AITags: []string{"cli"}, Topics: []string{"go"}, AISearchText: "一个命令行工具", AIPlatforms: []string{"cli"}, AnalyzedAt: &now, StargazersCount: 500},
	}, nil
}
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
func (m *mockStore) DeleteCategory(ctx context.Context, id string) (int, error) { return 0, nil }
func (m *mockStore) GetSyncState(ctx context.Context, key string) (string, error) { return "", nil }
func (m *mockStore) SetSyncState(ctx context.Context, key, value string) error { return nil }
func (m *mockStore) SaveSyncStats(ctx context.Context, stats *store.SyncStats) error { return nil }
func (m *mockStore) GetSyncStats(ctx context.Context) (*store.SyncStats, error) { return &store.SyncStats{}, nil }
func (m *mockStore) IncrementSyncCount(ctx context.Context) error { return nil }
func (m *mockStore) SearchFTS(ctx context.Context, query string, filters *store.SearchFilters) ([]*store.FTSResult, error) {
	return nil, nil
}
func (m *mockStore) RebuildFTSIndex(ctx context.Context) error { return nil }
func (m *mockStore) InsertVector(ctx context.Context, repoID int64, embedding []float64) error { return nil }
func (m *mockStore) SearchVectors(ctx context.Context, queryVec []float64, topK int, threshold float64) ([]store.VectorMatch, error) {
	return nil, nil
}
func (m *mockStore) DeleteVector(ctx context.Context, repoID int64) error { return nil }
func (m *mockStore) SetVectorIndexedAt(ctx context.Context, repoID int64, t time.Time) error { return nil }
func (m *mockStore) GetRepositoryByID(ctx context.Context, id int64) (*store.Repository, error) { return nil, nil }
func (m *mockStore) ListVectorUnindexed(ctx context.Context, limit int) ([]*store.Repository, error) { return nil, nil }
func (m *mockStore) EnsureVec0Dimension(ctx context.Context, dim int) error { return nil }
