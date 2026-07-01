package github

import (
	"context"
	"fmt"
	"time"

	"github.com/bartventer/httpcache"

	_ "github.com/bartventer/httpcache/store/memcache"

	gh "github.com/google/go-github/v71/github"
	"github.com/morehao/starman/internal/store"
	"golang.org/x/sync/errgroup"
)

type Repository = store.Repository

type Client struct {
	client *gh.Client
}

func New(token string) *Client {
	c := gh.NewClient(httpcache.NewClient("memcache://"))
	if token != "" {
		c = c.WithAuthToken(token)
	}
	return &Client{client: c}
}

func (c *Client) SetClient(ghc *gh.Client) { c.client = ghc }

const (
	perPage          = 100
	concurrentMax    = 90
	rateSafetyMargin = 10
)

func (c *Client) ListStarred(ctx context.Context, username string) ([]*Repository, error) {
	opts := &gh.ActivityListStarredOptions{
		ListOptions: gh.ListOptions{PerPage: perPage},
	}
	firstPage, resp, err := c.client.Activity.ListStarred(ctx, username, opts)
	if err != nil {
		return nil, fmt.Errorf("list starred first page: %w", err)
	}
	if resp.Rate.Remaining < rateSafetyMargin {
		return nil, fmt.Errorf("rate limit exceeded: %d remaining, reset at %s", resp.Rate.Remaining, resp.Rate.Reset.Format(time.RFC3339))
	}

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
	for {
		repos, resp, err := c.client.Activity.ListStarred(ctx, username, opts)
		if err != nil {
			if resp != nil && resp.Rate.Remaining == 0 {
				wait := time.Until(resp.Rate.Reset.Time)
				if wait > 0 {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(wait):
						continue
					}
				}
			}
			return nil, fmt.Errorf("fetch page %d: %w", page, err)
		}
		return repos, nil
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
