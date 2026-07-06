package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gh "github.com/google/go-github/v71/github"
)

func mockGitHubServer(t *testing.T, pages [][]*gh.StarredRepository) *httptest.Server {
	t.Helper()
	pageCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/users/testuser/starred", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "5000")
		w.Header().Set("X-RateLimit-Reset", "9999999999")
		page := pageCount
		if page < len(pages) {
			if page < len(pages)-1 {
				w.Header().Set("Link", `<https://example.com/users/testuser/starred?page=2>; rel="next"`)
			}
			_ = json.NewEncoder(w).Encode(pages[page])
		}
		pageCount++
	})
	return httptest.NewServer(mux)
}

func newTestClient(server *httptest.Server, token string) *Client {
	c := gh.NewClient(server.Client())
	baseURL, _ := url.Parse(server.URL + "/")
	c.BaseURL = baseURL
	return &Client{client: c}
}

func TestListStarredSinglePage(t *testing.T) {
	starredAt, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	page := []*gh.StarredRepository{
		{StarredAt: &gh.Timestamp{Time: starredAt}, Repository: &gh.Repository{ID: gh.Ptr(int64(1)), FullName: gh.Ptr("owner/repo1"), Name: gh.Ptr("repo1"), HTMLURL: gh.Ptr("https://github.com/owner/repo1")}},
	}
	server := mockGitHubServer(t, [][]*gh.StarredRepository{page})
	defer server.Close()
	c := newTestClient(server, "")
	repos, err := c.ListStarred(context.Background(), "testuser")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(repos))
	}
	if repos[0].FullName != "owner/repo1" {
		t.Fatalf("expected owner/repo1, got %s", repos[0].FullName)
	}
}
