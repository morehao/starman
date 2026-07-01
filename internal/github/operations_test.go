package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gh "github.com/google/go-github/v71/github"
)

func mockOpsServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	c := gh.NewClient(server.Client())
	baseURL, _ := url.Parse(server.URL + "/")
	c.BaseURL = baseURL
	return server, &Client{client: c}
}

func TestGetReadme(t *testing.T) {
	readmeContent := "# Test Repo\nHello"
	encoded := base64.StdEncoding.EncodeToString([]byte(readmeContent))
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/readme" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"content": encoded, "encoding": "base64"})
	})
	defer server.Close()
	got, err := c.GetReadme(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if got != readmeContent {
		t.Fatalf("expected %q, got %q", readmeContent, got)
	}
}

func TestListReleases(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Remaining", "5000")
		rels := []*gh.RepositoryRelease{
			{ID: gh.Ptr(int64(1)), TagName: gh.Ptr("v1.0.0"), Name: gh.Ptr("Release 1")},
			{ID: gh.Ptr(int64(2)), TagName: gh.Ptr("v2.0.0"), Name: gh.Ptr("Release 2")},
		}
		json.NewEncoder(w).Encode(rels)
	})
	defer server.Close()
	releases, err := c.ListReleases(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(releases))
	}
	if releases[0].TagName != "v1.0.0" {
		t.Fatalf("expected v1.0.0, got %s", releases[0].TagName)
	}
}

func TestStarUnstar(t *testing.T) {
	starCalled := false
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && r.URL.Path == "/user/starred/owner/repo" {
			starCalled = true
			w.WriteHeader(204)
			return
		}
		if r.Method == "DELETE" && r.URL.Path == "/user/starred/owner/repo" {
			w.WriteHeader(204)
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()
	if err := c.Star(context.Background(), "owner", "repo"); err != nil {
		t.Fatal(err)
	}
	if !starCalled {
		t.Fatal("star was not called")
	}
	if err := c.Unstar(context.Background(), "owner", "repo"); err != nil {
		t.Fatal(err)
	}
}
