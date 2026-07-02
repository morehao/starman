//go:build integration

package github

import (
	"context"
	"os"
	"testing"
	"time"
)

func testToken(t *testing.T) string {
	t.Helper()
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("STARMAN_GITHUB_TOKEN")
	}
	if token == "" {
		t.Skip("GITHUB_TOKEN or STARMAN_GITHUB_TOKEN not set, skipping integration test")
	}
	return token
}

func testUsername(t *testing.T) string {
	t.Helper()
	user := os.Getenv("GITHUB_USERNAME")
	if user == "" {
		t.Skip("GITHUB_USERNAME not set, skipping integration test")
	}
	return user
}

func TestListStarred_RealAPI(t *testing.T) {
	token := testToken(t)
	username := testUsername(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := New(token)
	repos, err := c.ListStarred(ctx, username)
	if err != nil {
		t.Fatalf("ListStarred failed: %v", err)
	}
	t.Logf("fetched %d starred repos for %s", len(repos), username)

	if len(repos) == 0 {
		t.Error("expected at least 1 starred repo")
	}

	for _, r := range repos {
		if r.FullName == "" {
			t.Error("repo has empty FullName")
		}
		if r.ID == 0 {
			t.Error("repo has zero ID")
		}
		if r.StarredAt == "" {
			t.Error("repo has empty StarredAt")
		}
	}
}

func TestListStarred_RealAPI_MultiPage(t *testing.T) {
	token := testToken(t)
	username := testUsername(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	c := New(token)
	repos, err := c.ListStarred(ctx, username)
	if err != nil {
		t.Fatalf("ListStarred failed: %v", err)
	}

	seen := make(map[string]bool)
	for _, r := range repos {
		if seen[r.FullName] {
			t.Errorf("duplicate repo: %s", r.FullName)
		}
		seen[r.FullName] = true
	}
	t.Logf("multipage sync: %d repos, no duplicates", len(repos))
}

func TestFetchStarredPage_WithRetry(t *testing.T) {
	token := testToken(t)
	username := testUsername(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := New(token)
	repos, err := c.fetchStarredPage(ctx, username, 1)
	if err != nil {
		t.Fatalf("fetchStarredPage failed: %v", err)
	}
	t.Logf("page 1: %d repos", len(repos))
}

func TestRateLimit_RealAPI(t *testing.T) {
	token := testToken(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := New(token)
	rate, err := c.RateLimit(ctx)
	if err != nil {
		t.Fatalf("RateLimit failed: %v", err)
	}
	t.Logf("rate limit: remaining=%d, limit=%d, reset=%s", rate.Remaining, rate.Limit, rate.Reset.Format(time.RFC3339))
}

func TestListReleasesIncremental_RealAPI(t *testing.T) {
	token := testToken(t)
	owner := os.Getenv("GITHUB_TEST_OWNER")
	repo := os.Getenv("GITHUB_TEST_REPO")
	if owner == "" || repo == "" {
		t.Skip("GITHUB_TEST_OWNER and GITHUB_TEST_REPO not set, skipping incremental release test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c := New(token)

	allReleases, err := c.ListReleases(ctx, owner, repo)
	if err != nil {
		t.Fatalf("ListReleases failed: %v", err)
	}
	t.Logf("full releases: %d for %s/%s", len(allReleases), owner, repo)

	if len(allReleases) > 0 {
		watermark := time.Now().Add(-365 * 24 * time.Hour)
		incReleases, err := c.ListReleasesIncremental(ctx, owner, repo, &watermark)
		if err != nil {
			t.Fatalf("ListReleasesIncremental failed: %v", err)
		}
		t.Logf("incremental releases (since 1y ago): %d", len(incReleases))

		if len(incReleases) < len(allReleases) {
			t.Logf("incremental correctly returned fewer releases than full (%d < %d)", len(incReleases), len(allReleases))
		}
	}

	watermark := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	emptyReleases, err := c.ListReleasesIncremental(ctx, owner, repo, &watermark)
	if err != nil {
		t.Fatalf("ListReleasesIncremental with future watermark failed: %v", err)
	}
	if len(emptyReleases) != 0 {
		t.Errorf("expected 0 releases with future watermark, got %d", len(emptyReleases))
	}
	t.Logf("future watermark test: %d releases (expected 0)", len(emptyReleases))
}
