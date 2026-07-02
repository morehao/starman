package release

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
)

type PullStats struct {
	Subscribed  int
	NewReleases int
	Errors      int
}

type Tracker struct {
	store store.Store
	gh    *github.Client
}

func NewTracker(s store.Store, gh *github.Client) *Tracker {
	return &Tracker{store: s, gh: gh}
}

func (t *Tracker) PullReleases(ctx context.Context) (*PullStats, error) {
	allRepos, err := t.store.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	var subscribed []*store.Repository
	for _, r := range allRepos {
		if r.SubscribedReleases {
			subscribed = append(subscribed, r)
		}
	}
	stats := &PullStats{Subscribed: len(subscribed)}

	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, repo := range subscribed {
		select {
		case <-ctx.Done():
			wg.Wait()
			return stats, ctx.Err()
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(r *store.Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			n, err := t.pullOne(ctx, r)
			if err != nil {
				mu.Lock()
				stats.Errors++
				mu.Unlock()
				return
			}
			mu.Lock()
			stats.NewReleases += n
			mu.Unlock()
		}(repo)
	}
	wg.Wait()
	return stats, nil
}

func (t *Tracker) pullOne(ctx context.Context, repo *store.Repository) (int, error) {
	parts := splitName(repo.FullName)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid full name: %s", repo.FullName)
	}

	var releases []*store.Release
	var err error
	if repo.LastReleaseFetch == nil {
		releases, err = t.gh.ListReleases(ctx, parts[0], parts[1])
	} else {
		releases, err = t.gh.ListReleasesIncremental(ctx, parts[0], parts[1], repo.LastReleaseFetch)
	}
	if err != nil {
		return 0, err
	}

	newCount := 0
	for _, rel := range releases {
		rel.RepoID = repo.ID
		if err := t.store.UpsertRelease(ctx, rel); err != nil {
			return newCount, err
		}
		newCount++
	}

	if len(releases) > 0 {
		latest := releases[0].PublishedAt
		pubTime, err := time.Parse(time.RFC3339, latest)
		if err == nil {
			t.store.UpdateReleaseWatermark(ctx, repo.ID, pubTime)
		}
	}
	return newCount, nil
}

func splitName(fullName string) []string {
	for i := 0; i < len(fullName); i++ {
		if fullName[i] == '/' {
			return []string{fullName[:i], fullName[i+1:]}
		}
	}
	return nil
}
