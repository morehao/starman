package ai

import (
	"context"
	"fmt"
	"sync"

	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
)

type BatchOpts struct {
	Force      bool
	Limit      int
	OnProgress func(done, total int, curName string)
}

type BatchAnalyzer struct {
	svc         *Service
	store       store.Store
	gh          *github.Client
	concurrency int
}

func NewBatchAnalyzer(svc *Service, s store.Store, gh *github.Client, concurrency int) *BatchAnalyzer {
	if concurrency < 1 {
		concurrency = 1
	}
	return &BatchAnalyzer{svc: svc, store: s, gh: gh, concurrency: concurrency}
}

type BatchResult struct {
	Total   int
	Success int
	Failed  int
}

func (b *BatchAnalyzer) Run(ctx context.Context, repos []*store.Repository, opts BatchOpts) (*BatchResult, error) {
	toAnalyze := repos
	if !opts.Force {
		filtered := make([]*store.Repository, 0, len(repos))
		for _, r := range repos {
			if r.AnalyzedAt == nil {
				filtered = append(filtered, r)
			}
		}
		toAnalyze = filtered
	}
	if opts.Limit > 0 && opts.Limit < len(toAnalyze) {
		toAnalyze = toAnalyze[:opts.Limit]
	}

	cats, err := b.store.ListCategories(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	sem := make(chan struct{}, b.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	result := &BatchResult{Total: len(toAnalyze)}
	done := 0

	for _, repo := range toAnalyze {
		select {
		case <-ctx.Done():
			wg.Wait()
			return result, ctx.Err()
		case sem <- struct{}{}:
		}
		wg.Add(1)
		go func(r *store.Repository) {
			defer wg.Done()
			defer func() { <-sem }()
			if opts.OnProgress != nil {
				mu.Lock()
				done++
				opts.OnProgress(done, result.Total, r.FullName)
				mu.Unlock()
			}
			if err := b.analyzeOne(ctx, r, cats); err != nil {
				mu.Lock()
				result.Failed++
				mu.Unlock()
				return
			}
			mu.Lock()
			result.Success++
			mu.Unlock()
		}(repo)
	}
	wg.Wait()
	return result, nil
}

func (b *BatchAnalyzer) analyzeOne(ctx context.Context, repo *store.Repository, cats []*store.Category) error {
	var readme string
	parts := splitFullName(repo.FullName)
	if len(parts) == 2 {
		r, err := b.gh.GetReadme(ctx, parts[0], parts[1])
		if err == nil {
			readme = r
		}
	}
	result, err := b.svc.AnalyzeRepository(ctx, repo, readme, cats)
	if err != nil {
		_ = b.store.UpdateCustomFields(ctx, repo.ID, &store.CustomFields{})
		return err
	}
	category := ResolveCategory(repo, result.Tags, cats)
	aiResult := &store.AIResult{
		Summary:   result.Summary,
		Tags:      result.Tags,
		Platforms: result.Platforms,
		Category:  category,
	}
	return b.store.UpdateAIResult(ctx, repo.ID, aiResult)
}

func splitFullName(fullName string) []string {
	for i := 0; i < len(fullName); i++ {
		if fullName[i] == '/' {
			return []string{fullName[:i], fullName[i+1:]}
		}
	}
	return nil
}
