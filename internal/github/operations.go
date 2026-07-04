package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/morehao/starman/internal/store"
	gh "github.com/google/go-github/v71/github"
)

type Release = store.Release
type ReleaseAsset = store.ReleaseAsset

func (c *Client) Star(ctx context.Context, owner, repo string) error {
	_, err := c.client.Activity.Star(ctx, owner, repo)
	return err
}

func (c *Client) Unstar(ctx context.Context, owner, repo string) error {
	_, err := c.client.Activity.Unstar(ctx, owner, repo)
	return err
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	r, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	return convertRepo(r), nil
}

func (c *Client) ListReleases(ctx context.Context, owner, repo string) ([]*Release, error) {
	opts := &gh.ListOptions{PerPage: 100}
	var all []*Release
	for {
		releases, resp, err := c.client.Repositories.ListReleases(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list releases %s/%s: %w", owner, repo, err)
		}
		for _, rel := range releases {
			all = append(all, convertRelease(rel, owner+"/"+repo))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return all, nil
}

func (c *Client) ListReleasesIncremental(ctx context.Context, owner, repo string, watermark *time.Time) ([]*Release, error) {
	opts := &gh.ListOptions{PerPage: perPage}
	var allReleases []*Release

	for {
		ghReleases, resp, err := c.client.Repositories.ListReleases(ctx, owner, repo, &gh.ListOptions{
			Page:    opts.Page,
			PerPage: opts.PerPage,
		})
		if err != nil {
			return nil, fmt.Errorf("list releases page %d: %w", opts.Page, err)
		}

		for _, ghRel := range ghReleases {
			rel := convertRelease(ghRel, owner+"/"+repo)
			pubTime, err := time.Parse(time.RFC3339, rel.PublishedAt)
			if err == nil && watermark != nil && !pubTime.After(*watermark) {
				return allReleases, nil
			}
			allReleases = append(allReleases, rel)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allReleases, nil
}

func (c *Client) GetReadme(ctx context.Context, owner, repo string) (string, error) {
	readme, _, err := c.client.Repositories.GetReadme(ctx, owner, repo, nil)
	if err != nil {
		return "", fmt.Errorf("get readme %s/%s: %w", owner, repo, err)
	}
	content, err := readme.GetContent()
	if err != nil {
		return "", fmt.Errorf("decode readme: %w", err)
	}
	return content, nil
}

func (c *Client) UpdateReadmeFile(ctx context.Context, owner, repo, content, message string) error {
	_, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return fmt.Errorf("repo %s/%s not accessible: %w", owner, repo, err)
	}
	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, "README.md", nil)
	if err != nil && resp != nil && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("get readme contents: %w", err)
	}
	if fileContent == nil {
		_, _, err = c.client.Repositories.CreateFile(ctx, owner, repo, "README.md", &gh.RepositoryContentFileOptions{
			Message: gh.Ptr(message),
			Content: []byte(content),
		})
		if err != nil {
			return fmt.Errorf("create readme: %w", err)
		}
	} else {
		_, _, err = c.client.Repositories.UpdateFile(ctx, owner, repo, "README.md", &gh.RepositoryContentFileOptions{
			Message: gh.Ptr(message),
			Content: []byte(content),
			SHA:     fileContent.SHA,
		})
		if err != nil {
			return fmt.Errorf("update readme: %w", err)
		}
	}
	return nil
}

func (c *Client) CommitFile(ctx context.Context, owner, repo, path string, content []byte, message string) error {
	if len(content) > 1*1024*1024 {
		return fmt.Errorf("file too large (%d bytes) for GitHub Contents API (max 1MB); use 'starman backup webdav --push'", len(content))
	}
	if _, _, err := c.client.Repositories.Get(ctx, owner, repo); err != nil {
		return fmt.Errorf("repo %s/%s not accessible: %w", owner, repo, err)
	}

	fileContent, _, resp, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil && resp != nil && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("get contents: %w", err)
	}

	opts := &gh.RepositoryContentFileOptions{
		Message: gh.Ptr(message),
		Content: content,
	}
	if fileContent != nil {
		opts.SHA = fileContent.SHA
	}

	if fileContent == nil {
		_, _, err = c.client.Repositories.CreateFile(ctx, owner, repo, path, opts)
	} else {
		_, _, err = c.client.Repositories.UpdateFile(ctx, owner, repo, path, opts)
	}
	if err != nil {
		return fmt.Errorf("commit file: %w", err)
	}
	return nil
}

type Rate struct {
	Remaining int
	Reset     time.Time
	Limit     int
}

func (c *Client) RateLimit(ctx context.Context) (*Rate, error) {
	rl, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &Rate{Remaining: rl.Core.Remaining, Reset: rl.Core.Reset.Time, Limit: rl.Core.Limit}, nil
}

func convertRepo(r *gh.Repository) *Repository {
	var topics []string
	if r.Topics != nil {
		topics = r.Topics
	}
	var repoUpdatedAt string
	if r.UpdatedAt != nil {
		repoUpdatedAt = r.UpdatedAt.Format(time.RFC3339)
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
		RepoUpdatedAt:   repoUpdatedAt,
	}
}

var readmeVariantRe = regexp.MustCompile(`(?i)^readme([._-]?([a-z]{2})(?:-[a-z]{2})?)?\.(md|txt|markdown|rst)$`)

func (c *Client) ListReadmeVariants(ctx context.Context, owner, repo string) ([]string, error) {
	_, contents, _, err := c.client.Repositories.GetContents(ctx, owner, repo, "", nil)
	if err != nil {
		return nil, fmt.Errorf("list contents %s/%s: %w", owner, repo, err)
	}
	var variants []string
	for _, content := range contents {
		name := content.GetName()
		if isReadmeVariant(name) {
			variants = append(variants, name)
		}
	}
	return variants, nil
}

func isReadmeVariant(name string) bool {
	lower := strings.ToLower(name)
	if lower == "readme" {
		return true
	}
	return readmeVariantRe.MatchString(lower)
}

func (c *Client) GetContentFile(ctx context.Context, owner, repo, path string) (string, error) {
	content, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return "", fmt.Errorf("get content %s/%s/%s: %w", owner, repo, path, err)
	}
	encoded, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("get encoded content %s/%s/%s: %w", owner, repo, path, err)
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return encoded, nil
	}
	return string(decoded), nil
}

func (c *Client) SearchRepositories(ctx context.Context, query string) ([]*Repository, error) {
	opts := &gh.SearchOptions{
		Sort:        "stars",
		Order:       "desc",
		ListOptions: gh.ListOptions{PerPage: 50},
	}
	result, _, err := c.client.Search.Repositories(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("search repositories: %w", err)
	}
	var repos []*Repository
	for _, r := range result.Repositories {
		repos = append(repos, convertRepo(r))
	}
	return repos, nil
}

func convertRelease(rel *gh.RepositoryRelease, repoFullName string) *Release {
	var assets []ReleaseAsset
	for _, a := range rel.Assets {
		assets = append(assets, ReleaseAsset{
			Name:        a.GetName(),
			URL:         a.GetBrowserDownloadURL(),
			Size:        int64(a.GetSize()),
			ContentType: a.GetContentType(),
		})
	}
	repoID := int64(0)
	return &Release{
		ID:           rel.GetID(),
		RepoID:       repoID,
		RepoFullName: repoFullName,
		TagName:      rel.GetTagName(),
		Name:         rel.GetName(),
		Body:         rel.GetBody(),
		HTMLURL:      rel.GetHTMLURL(),
		PublishedAt:  rel.GetPublishedAt().Format(time.RFC3339),
		IsPrerelease: rel.GetPrerelease(),
		IsDraft:      rel.GetDraft(),
		Assets:       assets,
	}
}
