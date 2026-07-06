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
		_ = json.NewEncoder(w).Encode(map[string]any{"content": encoded, "encoding": "base64"})
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
			_ = json.NewEncoder(w).Encode([]*gh.RepositoryContent{
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
		_ = json.NewEncoder(w).Encode(&gh.RepositoryContent{
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
	var blobCreated, treeCreated, commitCreated, refUpdated bool
	baseSha := "abc123"
	treeSha := "tree456"
	blobSha := "blob789"
	commitSha := "commit000"

	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/repo"})
		case r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/blobs":
			blobCreated = true
			_ = json.NewEncoder(w).Encode(&gh.Blob{SHA: gh.Ptr(blobSha)})
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo/git/ref/heads/main":
			_ = json.NewEncoder(w).Encode(&gh.Reference{
				Ref:    gh.Ptr("refs/heads/main"),
				Object: &gh.GitObject{Type: gh.Ptr("commit"), SHA: gh.Ptr(baseSha)},
			})
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo/git/commits/"+baseSha:
			_ = json.NewEncoder(w).Encode(&gh.Commit{
				SHA:     gh.Ptr(baseSha),
				Tree:    &gh.Tree{SHA: gh.Ptr(treeSha)},
				Parents: []*gh.Commit{{SHA: gh.Ptr("parent")}},
			})
		case r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/trees":
			treeCreated = true
			_ = json.NewEncoder(w).Encode(&gh.Tree{SHA: gh.Ptr(treeSha)})
		case r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/commits":
			commitCreated = true
			_ = json.NewEncoder(w).Encode(&gh.Commit{SHA: gh.Ptr(commitSha)})
		case r.Method == "PATCH" && r.URL.Path == "/repos/owner/repo/git/refs/heads/main":
			refUpdated = true
			w.WriteHeader(200)
			_ = json.NewEncoder(w).Encode(&gh.Reference{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	content := []byte("backup data")
	err := c.CommitFile(context.Background(), "owner", "repo", "starman-backup/starman.db", content, "backup starman data")
	if err != nil {
		t.Fatal(err)
	}
	if !blobCreated {
		t.Fatal("blob was not created")
	}
	if !treeCreated {
		t.Fatal("tree was not created")
	}
	if !commitCreated {
		t.Fatal("commit was not created")
	}
	if !refUpdated {
		t.Fatal("ref was not updated")
	}
}

func TestCommitFile_MasterBranch(t *testing.T) {
	baseSha := "abc123"
	treeSha := "tree456"
	blobSha := "blob789"
	commitSha := "commit000"

	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "full_name": "owner/repo"})
		case r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/blobs":
			_ = json.NewEncoder(w).Encode(&gh.Blob{SHA: gh.Ptr(blobSha)})
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo/git/ref/heads/main":
			w.WriteHeader(404)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo/git/ref/heads/master":
			_ = json.NewEncoder(w).Encode(&gh.Reference{
				Ref:    gh.Ptr("refs/heads/master"),
				Object: &gh.GitObject{Type: gh.Ptr("commit"), SHA: gh.Ptr(baseSha)},
			})
		case r.Method == "GET" && r.URL.Path == "/repos/owner/repo/git/commits/"+baseSha:
			_ = json.NewEncoder(w).Encode(&gh.Commit{
				SHA:     gh.Ptr(baseSha),
				Tree:    &gh.Tree{SHA: gh.Ptr(treeSha)},
				Parents: []*gh.Commit{{SHA: gh.Ptr("parent")}},
			})
		case r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/trees":
			_ = json.NewEncoder(w).Encode(&gh.Tree{SHA: gh.Ptr(treeSha)})
		case r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/commits":
			_ = json.NewEncoder(w).Encode(&gh.Commit{SHA: gh.Ptr(commitSha)})
		case r.Method == "PATCH" && r.URL.Path == "/repos/owner/repo/git/refs/heads/master":
			w.WriteHeader(200)
			_ = json.NewEncoder(w).Encode(&gh.Reference{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := c.CommitFile(context.Background(), "owner", "repo", "starman-backup/starman.db", []byte("backup"), "backup")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCommitFile_RepoNotFound(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo" {
			w.WriteHeader(404)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "Not Found"})
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := c.CommitFile(context.Background(), "owner", "repo", "starman-backup/starman.db", []byte(`{}`), "test")
	if err == nil {
		t.Fatal("expected error for non-existent repo")
	}
}

func TestSearchRepositories(t *testing.T) {
	server, c := mockOpsServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/search/repositories") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(&gh.RepositoriesSearchResult{
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
