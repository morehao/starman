package store

import (
	"context"
	"testing"
	"time"
)

func sampleRepo(id int64, name string) *Repository {
	return &Repository{
		ID:              id,
		FullName:        name,
		Name:            name,
		Description:     "test desc",
		URL:             "https://github.com/" + name,
		Language:        "Go",
		StargazersCount: 10,
		Topics:          []string{"go", "cli"},
		OwnerLogin:      "owner",
		StarredAt:       "2026-01-01T00:00:00Z",
		RepoUpdatedAt:   "2026-01-15T00:00:00Z",
	}
}

func TestUpsertAndListRepositories(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepositories(ctx, []*Repository{
		sampleRepo(1, "owner/repo1"),
		sampleRepo(2, "owner/repo2"),
	}); err != nil {
		t.Fatal(err)
	}
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "owner/repo1" {
		t.Fatalf("unexpected first repo: %s", repos[0].FullName)
	}
	if repos[0].RepoUpdatedAt != "2026-01-15T00:00:00Z" {
		t.Fatalf("expected repo_updated_at '2026-01-15T00:00:00Z', got %q", repos[0].RepoUpdatedAt)
	}
}

func TestUpsertReposOnSyncPreservesAIFields(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	r1 := sampleRepo(1, "owner/repo1")
	if err := s.UpsertRepository(ctx, r1); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	err := s.UpdateAIResult(ctx, 1, &AIResult{Summary: "AI summary", Tags: []string{"tag1"}, Category: "dev-tools"})
	if err != nil {
		t.Fatal(err)
	}
	_ = now

	r1Sync := sampleRepo(1, "owner/repo1")
	r1Sync.Description = "updated desc"
	r1Sync.StargazersCount = 20
	r1Sync.RepoUpdatedAt = "2026-07-04T12:00:00Z"
	if err := s.UpsertReposOnSync(ctx, []*Repository{r1Sync}, false); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Description != "updated desc" {
		t.Fatalf("expected updated desc, got %s", got.Description)
	}
	if got.StargazersCount != 20 {
		t.Fatalf("expected 20 stars, got %d", got.StargazersCount)
	}
	if got.AISummary != "AI summary" {
		t.Fatalf("AI summary should be preserved, got %s", got.AISummary)
	}
	if got.AICategory != "dev-tools" {
		t.Fatalf("AI category should be preserved, got %s", got.AICategory)
	}
	if got.RepoUpdatedAt != "2026-07-04T12:00:00Z" {
		t.Fatalf("RepoUpdatedAt should be overwritten by sync, got %q", got.RepoUpdatedAt)
	}
}

func TestUpsertReposOnSyncFullSyncDeletes(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepository(ctx, sampleRepo(2, "owner/repo2")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertReposOnSync(ctx, []*Repository{sampleRepo(1, "owner/repo1")}, true); err != nil {
		t.Fatal(err)
	}
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("expected 1 repo after full sync, got %d", len(repos))
	}
	if repos[0].FullName != "owner/repo1" {
		t.Fatalf("expected owner/repo1, got %s", repos[0].FullName)
	}
}

func TestUpdateAIResult(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	err := s.UpdateAIResult(ctx, 1, &AIResult{Summary: "summary", Tags: []string{"t1", "t2"}, Platforms: []string{"web"}, Category: "web-app"})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if r.AISummary != "summary" {
		t.Fatalf("expected summary, got %s", r.AISummary)
	}
	if len(r.AITags) != 2 || r.AITags[0] != "t1" {
		t.Fatalf("unexpected tags: %v", r.AITags)
	}
	if r.AICategory != "web-app" {
		t.Fatalf("expected web-app, got %s", r.AICategory)
	}
	if r.AnalyzedAt == nil {
		t.Fatal("expected analyzed_at to be set")
	}
	if r.AnalysisFailed {
		t.Fatal("analysis_failed should be false after success")
	}
}

func TestListUnanalyzed(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepository(ctx, sampleRepo(2, "owner/repo2")); err != nil {
		t.Fatal(err)
	}
	err := s.UpdateAIResult(ctx, 1, &AIResult{Summary: "x", Category: "others"})
	if err != nil {
		t.Fatal(err)
	}
	unanalyzed, err := s.ListUnanalyzed(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unanalyzed) != 1 || unanalyzed[0].FullName != "owner/repo2" {
		t.Fatalf("expected only owner/repo2 unanalyzed, got %v", unanalyzed)
	}
}

func TestListByCategory(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepository(ctx, sampleRepo(2, "owner/repo2")); err != nil {
		t.Fatal(err)
	}
	err := s.UpdateAIResult(ctx, 1, &AIResult{Summary: "x", Category: "dev-tools"})
	if err != nil {
		t.Fatal(err)
	}
	err = s.UpdateAIResult(ctx, 2, &AIResult{Summary: "y", Category: "web-app"})
	if err != nil {
		t.Fatal(err)
	}
	devTools, err := s.ListByCategory(ctx, "dev-tools")
	if err != nil {
		t.Fatal(err)
	}
	if len(devTools) != 1 || devTools[0].FullName != "owner/repo1" {
		t.Fatalf("expected owner/repo1 in dev-tools, got %v", devTools)
	}
}

func TestSetAnalysisFailed(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAnalysisFailed(ctx, 1, true); err != nil {
		t.Fatal(err)
	}
	r, err := s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if !r.AnalysisFailed {
		t.Fatal("expected analysis_failed = true")
	}
	if r.AISummary != "" {
		t.Fatalf("analysis failure should not wipe AI fields, got summary %q", r.AISummary)
	}
	if err := s.SetAnalysisFailed(ctx, 1, false); err != nil {
		t.Fatal(err)
	}
	r, err = s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if r.AnalysisFailed {
		t.Fatal("expected analysis_failed = false")
	}
}

func TestUpsertReposTouchOnly(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	r1 := sampleRepo(1, "owner/repo1")
	if err := s.UpsertRepository(ctx, r1); err != nil {
		t.Fatal(err)
	}

	r1.RepoUpdatedAt = "2026-07-04T12:00:00Z"
	if err := s.UpsertReposTouchOnly(ctx, []*Repository{r1}); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if got.RepoUpdatedAt != "2026-07-04T12:00:00Z" {
		t.Fatalf("expected repo_updated_at '2026-07-04T12:00:00Z', got %q", got.RepoUpdatedAt)
	}
	if got.Description != "test desc" {
		t.Fatalf("touch should not modify description, got %q", got.Description)
	}
	if got.StargazersCount != 10 {
		t.Fatalf("touch should not modify stargazers_count, got %d", got.StargazersCount)
	}
	if got.StarredAt != r1.StarredAt {
		t.Fatalf("touch should not modify starred_at, got %q", got.StarredAt)
	}
}

func TestDeleteAllRepositories(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRepository(ctx, sampleRepo(2, "owner/repo2")); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteAllRepositories(ctx); err != nil {
		t.Fatal(err)
	}
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Fatalf("expected 0 repos after delete all, got %d", len(repos))
	}
}
