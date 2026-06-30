package store

import (
	"context"
	"testing"
	"time"
)

func TestReleaseCRUD(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	rel := &Release{
		ID:           100,
		RepoID:       1,
		RepoFullName: "owner/repo1",
		TagName:      "v1.0.0",
		Name:         "First release",
		HTMLURL:      "https://github.com/owner/repo1/releases/tag/v1.0.0",
		PublishedAt:  "2026-01-15T10:00:00Z",
		Assets: []ReleaseAsset{
			{Name: "bin.tar.gz", URL: "https://example.com/bin.tar.gz", Size: 1024, ContentType: "application/gzip"},
		},
	}
	if err := s.UpsertRelease(ctx, rel); err != nil {
		t.Fatal(err)
	}
	unread, err := s.ListUnreadReleases(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(unread) != 1 {
		t.Fatalf("expected 1 unread release, got %d", len(unread))
	}
	if unread[0].TagName != "v1.0.0" {
		t.Fatalf("expected v1.0.0, got %s", unread[0].TagName)
	}
	if len(unread[0].Assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(unread[0].Assets))
	}
	if err := s.MarkReleaseRead(ctx, 100); err != nil {
		t.Fatal(err)
	}
	unread, _ = s.ListUnreadReleases(ctx)
	if len(unread) != 0 {
		t.Fatalf("expected 0 unread after mark read, got %d", len(unread))
	}
	byRepo, err := s.ListReleasesByRepo(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if len(byRepo) != 1 {
		t.Fatalf("expected 1 release by repo, got %d", len(byRepo))
	}
}

func TestSetReleaseSubscription(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	if err := s.SetReleaseSubscription(ctx, "owner/repo1", true); err != nil {
		t.Fatal(err)
	}
	r, err := s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if !r.SubscribedReleases {
		t.Fatal("expected subscribed_releases = true")
	}
	err = s.SetReleaseSubscription(ctx, "nonexistent/repo", true)
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
}

func TestUpdateReleaseWatermark(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := s.UpdateReleaseWatermark(ctx, 1, now); err != nil {
		t.Fatal(err)
	}
	r, err := s.GetRepository(ctx, "owner/repo1")
	if err != nil {
		t.Fatal(err)
	}
	if r.LastReleaseFetch == nil {
		t.Fatal("expected last_release_fetch to be set")
	}
}

func TestMarkAllReleasesRead(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.UpsertRepository(ctx, sampleRepo(1, "owner/repo1")); err != nil {
		t.Fatal(err)
	}
	s.UpsertRelease(ctx, &Release{ID: 1, RepoID: 1, RepoFullName: "owner/repo1", TagName: "v1", PublishedAt: "2026-01-01T00:00:00Z"})
	s.UpsertRelease(ctx, &Release{ID: 2, RepoID: 1, RepoFullName: "owner/repo1", TagName: "v2", PublishedAt: "2026-02-01T00:00:00Z"})
	if err := s.MarkAllReleasesRead(ctx); err != nil {
		t.Fatal(err)
	}
	unread, _ := s.ListUnreadReleases(ctx)
	if len(unread) != 0 {
		t.Fatalf("expected 0 unread after mark all, got %d", len(unread))
	}
}
