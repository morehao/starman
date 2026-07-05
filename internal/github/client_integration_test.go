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


