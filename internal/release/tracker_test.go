package release

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gh "github.com/google/go-github/v71/github"
	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
)

type mockStore struct {
	repos []*store.Repository
	store.Store
}

func (m *mockStore) ListRepositories(ctx context.Context) ([]*store.Repository, error) {
	return m.repos, nil
}

func (m *mockStore) UpsertRelease(ctx context.Context, r *store.Release) error {
	return nil
}

func (m *mockStore) UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error {
	return nil
}

func TestPullReleasesIncremental(t *testing.T) {
	watermark := time.Now().Add(-24 * time.Hour)
	repos := []*store.Repository{
		{ID: 1, FullName: "owner/repo1", SubscribedReleases: true, LastReleaseFetch: &watermark},
	}
	ms := &mockStore{repos: repos}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "5000")
		rels := []*gh.RepositoryRelease{
			{ID: gh.Ptr(int64(1)), TagName: gh.Ptr("v1.0.0"), PublishedAt: &gh.Timestamp{Time: time.Now().Add(-48 * time.Hour)}},
			{ID: gh.Ptr(int64(2)), TagName: gh.Ptr("v2.0.0"), PublishedAt: &gh.Timestamp{Time: time.Now()}},
		}
		json.NewEncoder(w).Encode(rels)
	}))
	defer server.Close()

	c := gh.NewClient(server.Client())
	baseURL, _ := url.Parse(server.URL + "/")
	c.BaseURL = baseURL

	gc := github.New("")
	gc.SetClient(c)
	tracker := NewTracker(ms, gc)

	stats, err := tracker.PullReleases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Subscribed != 1 {
		t.Fatalf("expected 1 subscribed, got %d", stats.Subscribed)
	}
	if stats.NewReleases != 1 {
		t.Fatalf("expected 1 new release (after watermark), got %d", stats.NewReleases)
	}
}
