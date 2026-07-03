package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	gh "github.com/google/go-github/v71/github"
	"github.com/morehao/starman/internal/store"
	"golang.org/x/sync/errgroup"
)

type Repository = store.Repository

type Client struct {
	client *gh.Client
}

func New(token string) *Client {
	c := gh.NewClient(&http.Client{})
	if token != "" {
		c = c.WithAuthToken(token)
	}
	return &Client{client: c}
}

func (c *Client) SetClient(ghc *gh.Client) { c.client = ghc }

const (
	perPage          = 100
	concurrentMax    = 30
	rateSafetyMargin = 20
)

func (c *Client) ListStarred(ctx context.Context, username string) ([]*Repository, error) {
	opts := &gh.ActivityListStarredOptions{
		ListOptions: gh.ListOptions{PerPage: perPage},
	}
	firstPage, resp, err := c.client.Activity.ListStarred(ctx, username, opts)
	if err != nil {
		return nil, fmt.Errorf("list starred first page: %w", err)
	}

	var rateRemaining atomic.Int64
	rateRemaining.Store(int64(resp.Rate.Remaining))
	rateReset := resp.Rate.Reset.Time
	rateThreshold := int64(rateSafetyMargin)

	if rateRemaining.Load() < rateThreshold {
		if err := waitForRateReset(ctx, rateReset); err != nil {
			return nil, err
		}
		rateRemaining.Store(5000)
	}
	rateRemaining.Add(-1)

	totalPages := resp.LastPage
	if totalPages < 1 {
		totalPages = 1
	}

	pages := make([][]*gh.StarredRepository, totalPages)
	pages[0] = firstPage

	if totalPages > 1 {
		eg, ctx := errgroup.WithContext(ctx)
		eg.SetLimit(concurrentMax)
		for page := 2; page <= totalPages; page++ {
			page := page
			eg.Go(func() error {
				if err := checkAndWaitRateLimit(ctx, &rateRemaining, rateReset, rateThreshold); err != nil {
					return err
				}
				rateRemaining.Add(-1)

				repos, err := c.fetchStarredPage(ctx, username, page)
				if err != nil {
					return err
				}
				pages[page-1] = repos
				return nil
			})
		}
		if err := eg.Wait(); err != nil {
			return nil, err
		}
	}

	repos := make([]*Repository, 0, len(firstPage)*totalPages)
	for _, page := range pages {
		for _, sr := range page {
			repos = append(repos, convertStarred(sr))
		}
	}
	return repos, nil
}

func (c *Client) fetchStarredPage(ctx context.Context, username string, page int) ([]*gh.StarredRepository, error) {
	opts := &gh.ActivityListStarredOptions{
		ListOptions: gh.ListOptions{PerPage: perPage, Page: page},
	}

	var repos []*gh.StarredRepository
	err := retryWithBackoff(ctx, 3, time.Second, func() error {
		result, resp, err := c.client.Activity.ListStarred(ctx, username, opts)
		if err != nil {
			if resp != nil && resp.Rate.Remaining == 0 {
				return fmt.Errorf("rate limit exhausted: %w", err)
			}
			return err
		}
		repos = result
		return nil
	})
	return repos, err
}

func checkAndWaitRateLimit(ctx context.Context, remaining *atomic.Int64, resetTime time.Time, threshold int64) error {
	if remaining.Load() < threshold {
		return waitForRateReset(ctx, resetTime)
	}
	return nil
}

func waitForRateReset(ctx context.Context, resetTime time.Time) error {
	waitDur := time.Until(resetTime) + time.Second
	if waitDur <= 0 {
		return nil
	}
	log.Printf("[starman] Rate limit low, waiting %v until reset...", waitDur.Round(time.Second))
	return sleep(ctx, waitDur)
}

func retryWithBackoff(ctx context.Context, maxRetries int, baseDelay time.Duration, fn func() error) error {
	for i := 0; i <= maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		if i == maxRetries {
			return fmt.Errorf("retry exhausted (%d attempts): %w", maxRetries+1, err)
		}

		if !isRetryable(err) {
			return err
		}

		delay := baseDelay * time.Duration(1<<i)
		log.Printf("[starman] Request failed, retrying in %v (attempt %d/%d): %v",
			delay, i+1, maxRetries, err)
		if sleepErr := sleep(ctx, delay); sleepErr != nil {
			return sleepErr
		}
	}
	return nil
}

func isRetryable(err error) bool {
	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) {
		code := ghErr.Response.StatusCode
		if code >= 500 {
			return true
		}
		if code == 403 {
			return true
		}
		return false
	}
	return true
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func convertStarred(sr *gh.StarredRepository) *Repository {
	r := sr.GetRepository()
	var topics []string
	if r.Topics != nil {
		topics = r.Topics
	}
	return &Repository{
		ID:              r.GetID(),
		FullName:        r.GetFullName(),
		Name:            r.GetName(),
		Description:     r.GetDescription(),
		URL:             r.GetHTMLURL(),
		Language:        r.GetLanguage(),
		Homepage:        r.GetHomepage(),
		StargazersCount: r.GetStargazersCount(),
		ForksCount:      r.GetForksCount(),
		Topics:          topics,
		OwnerLogin:      r.GetOwner().GetLogin(),
		OwnerAvatar:     r.GetOwner().GetAvatarURL(),
		StarredAt:       sr.GetStarredAt().Format(time.RFC3339),
	}
}
