package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestListReadmeVariants(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repos/owner/repo/contents") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*gh.RepositoryContent{
				{Name: gh.Ptr("README.md"), Type: gh.Ptr("file")},
				{Name: gh.Ptr("README_zh.md"), Type: gh.Ptr("file")},
				{Name: gh.Ptr("main.go"), Type: gh.Ptr("file")},
				{Name: gh.Ptr("docs"), Type: gh.Ptr("dir")},
			})
			return
		}
		http.Error(w, "not found", 404)
	})
	defer server.Close()
	variants, err := c.ListReadmeVariants(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(variants) != 2 {
		t.Fatalf("expected 2 README variants, got %d: %v", len(variants), variants)
	}
}

func TestGetContentFile(t *testing.T) {
	content := base64.StdEncoding.EncodeToString([]byte("# Hello World"))
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&gh.RepositoryContent{
			Content:  gh.Ptr(content),
			Encoding: gh.Ptr("base64"),
		})
	})
	defer server.Close()
	got, err := c.GetContentFile(context.Background(), "owner", "repo", "README_zh.md")
	if err != nil {
		t.Fatal(err)
	}
	if got != "# Hello World" {
		t.Fatalf("expected '# Hello World', got %s", got)
	}
}

func TestCommitFile_Create(t *testing.T) {
	var method, reqPath string
	var reqBody map[string]any
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/repos/owner/repo" {
			json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/repo"})
			return
		}
		if r.Method == "GET" && r.URL.Path == "/repos/owner/repo/contents/starman-backup/2025-07-03.json" {
			w.WriteHeader(404)
			return
		}
		method = r.Method
		reqPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&reqBody)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"content": map[string]any{}})
	})
	defer server.Close()

	content := []byte(`{"version":1,"repositories":[]}`)
	err := c.CommitFile(context.Background(), "owner", "repo", "starman-backup/2025-07-03.json", content, "backup starman data 2025-07-03")
	if err != nil {
		t.Fatal(err)
	}
	if method != "PUT" {
		t.Fatalf("expected PUT, got %s", method)
	}
	if reqPath != "/repos/owner/repo/contents/starman-backup/2025-07-03.json" {
		t.Fatalf("unexpected path: %s", reqPath)
	}
	if reqBody["message"] != "backup starman data 2025-07-03" {
		t.Fatalf("unexpected message: %v", reqBody["message"])
	}
}

func TestCommitFile_Update(t *testing.T) {
	var method string
	var reqBody map[string]any
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/repos/owner/repo" {
			json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/repo"})
			return
		}
		if r.Method == "GET" && r.URL.Path == "/repos/owner/repo/contents/starman-backup/2025-07-03.json" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"content":  base64.StdEncoding.EncodeToString([]byte("old")),
				"encoding": "base64",
				"sha":      "abc123",
			})
			return
		}
		method = r.Method
		json.NewDecoder(r.Body).Decode(&reqBody)
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{"content": map[string]any{}})
	})
	defer server.Close()

	err := c.CommitFile(context.Background(), "owner", "repo", "starman-backup/2025-07-03.json", []byte(`{"new":true}`), "backup starman data 2025-07-03")
	if err != nil {
		t.Fatal(err)
	}
	if method != "PUT" {
		t.Fatalf("expected PUT, got %s", method)
	}
	if reqBody["sha"] != "abc123" {
		t.Fatalf("expected sha abc123, got %v", reqBody["sha"])
	}
}

func TestCommitFile_RepoNotFound(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo" {
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := c.CommitFile(context.Background(), "owner", "repo", "starman-backup/test.json", []byte(`{}`), "test")
	if err == nil {
		t.Fatal("expected error for non-existent repo")
	}
}

func TestSearchRepositories(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/search/repositories") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&gh.RepositoriesSearchResult{
				Repositories: []*gh.Repository{
					{ID: gh.Ptr(int64(1)), FullName: gh.Ptr("owner/repo1"), Name: gh.Ptr("repo1"), StargazersCount: gh.Ptr(100)},
					{ID: gh.Ptr(int64(2)), FullName: gh.Ptr("owner/repo2"), Name: gh.Ptr("repo2"), StargazersCount: gh.Ptr(50)},
				},
			})
			return
		}
		http.Error(w, "not found", 404)
	})
	defer server.Close()
	repos, err := c.SearchRepositories(context.Background(), "stars:>1000")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "owner/repo1" {
		t.Fatalf("expected owner/repo1, got %s", repos[0].FullName)
	}
}
