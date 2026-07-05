package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	gh "github.com/google/go-github/v71/github"
)

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
	if _, _, err := c.client.Repositories.Get(ctx, owner, repo); err != nil {
		return fmt.Errorf("repo %s/%s not accessible: %w", owner, repo, err)
	}

	blob, _, err := c.client.Git.CreateBlob(ctx, owner, repo, &gh.Blob{
		Content:  gh.Ptr(string(content)),
		Encoding: gh.Ptr("utf-8"),
	})
	if err != nil {
		return fmt.Errorf("create blob: %w", err)
	}

	ref, _, err := c.client.Git.GetRef(ctx, owner, repo, "refs/heads/main")
	if err != nil {
		masterRef, _, masterErr := c.client.Git.GetRef(ctx, owner, repo, "refs/heads/master")
		if masterErr != nil {
			return fmt.Errorf("get ref: %w", err)
		}
		ref = masterRef
	}

	baseCommit, _, err := c.client.Git.GetCommit(ctx, owner, repo, ref.GetObject().GetSHA())
	if err != nil {
		return fmt.Errorf("get commit: %w", err)
	}

	tree, _, err := c.client.Git.CreateTree(ctx, owner, repo, baseCommit.GetTree().GetSHA(), []*gh.TreeEntry{{
		Path: gh.Ptr(path),
		Mode: gh.Ptr("100644"),
		Type: gh.Ptr("blob"),
		SHA:  blob.SHA,
	}})
	if err != nil {
		return fmt.Errorf("create tree: %w", err)
	}

	now := time.Now()
	newCommit, _, err := c.client.Git.CreateCommit(ctx, owner, repo, &gh.Commit{
		Message: gh.Ptr(message),
		Tree:    tree,
		Parents: []*gh.Commit{baseCommit},
		Author: &gh.CommitAuthor{
			Name:  gh.Ptr("starman"),
			Email: gh.Ptr("starman@users.noreply.github.com"),
			Date:  &gh.Timestamp{Time: now},
		},
	}, nil)
	if err != nil {
		return fmt.Errorf("create commit: %w", err)
	}

	_, _, err = c.client.Git.UpdateRef(ctx, owner, repo, &gh.Reference{
		Ref: gh.Ptr("refs/heads/" + ref.GetRef()[11:]),
		Object: &gh.GitObject{
			SHA: newCommit.SHA,
		},
	}, false)
	if err != nil {
		return fmt.Errorf("update ref: %w", err)
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


