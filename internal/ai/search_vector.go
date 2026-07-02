package ai

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
)

func (s *Service) vectorSearch(
	ctx context.Context,
	query string,
	st store.Store,
	opts SearchOpts,
) (*SearchResult, error) {

	embeddingQuery := hydeEnhance(ctx, s, query, opts.EnableHyDE)

	queryVectors, err := s.embeddingClient.Embed(ctx, []string{embeddingQuery})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(queryVectors) == 0 || len(queryVectors[0]) == 0 {
		return nil, fmt.Errorf("empty query vector")
	}

	if dim := s.embeddingClient.Dimension(); dim > 0 {
		if err := st.EnsureVec0Dimension(ctx, dim); err != nil {
			return nil, fmt.Errorf("ensure vec0 dimension: %w", err)
		}
	}

	matches, err := st.SearchVectors(ctx, queryVectors[0], 30, 0.35)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	if len(matches) == 0 {
		return &SearchResult{Hits: []*SearchHit{}}, nil
	}

	queryLower := strings.ToLower(query)
	scoreMap := make(map[int64]float64, len(matches))
	for _, m := range matches {
		repo, err := st.GetRepositoryByID(ctx, m.RepoID)
		if err != nil {
			continue
		}
		bonus := 0.0
		if strings.Contains(strings.ToLower(repo.FullName), queryLower) {
			bonus += 0.05
		}
		if strings.Contains(strings.ToLower(repo.Description), queryLower) {
			bonus += 0.03
		}
		allTags := append(append([]string{}, repo.AITags...), repo.Topics...)
		for _, tag := range allTags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				bonus += 0.02
				break
			}
		}
		scoreMap[m.RepoID] = m.Similarity + bonus
	}

	type scoredRepo struct {
		repo  *store.Repository
		score float64
	}
	var scored []scoredRepo
	for repoID, score := range scoreMap {
		repo, err := st.GetRepositoryByID(ctx, repoID)
		if err != nil {
			continue
		}
		scored = append(scored, scoredRepo{repo, score})
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })

	hits := make([]*SearchHit, len(scored))
	for i, sr := range scored {
		hits[i] = &SearchHit{Repo: sr.repo, Score: sr.score}
	}

	if opts.EnableRerank && s.hasAIConfig() && len(hits) > 0 {
		topK := opts.RerankTopK
		if topK <= 0 {
			topK = 30
		}
		topK = min(topK, len(hits))
		reranked, err := s.rerank(ctx, query, hits[:topK])
		if err == nil {
			hits = append(reranked, hits[topK:]...)
		}
	}

	// Apply CLI-side filter and sort (Language/Category/Platform/Tags/MinStars/MaxStars)
	hits = filterHits(hits, opts)
	sortHits(hits, opts.Sort)
	if opts.Limit > 0 && opts.Limit < len(hits) {
		hits = hits[:opts.Limit]
	}

	return &SearchResult{Hits: hits}, nil
}

func filterHits(hits []*SearchHit, opts SearchOpts) []*SearchHit {
	filtered := make([]*SearchHit, 0, len(hits))
	for _, h := range hits {
		if opts.Language != "" && !strings.EqualFold(h.Repo.Language, opts.Language) {
			continue
		}
		if opts.Category != "" {
			cat := h.Repo.CustomCategory
			if cat == "" {
				cat = h.Repo.AICategory
			}
			if !strings.EqualFold(cat, opts.Category) {
				continue
			}
		}
		if opts.Platform != "" {
			platMatch := false
			for _, p := range h.Repo.AIPlatforms {
				if strings.EqualFold(p, opts.Platform) {
					platMatch = true
					break
				}
			}
			if !platMatch {
				continue
			}
		}
		if opts.MinStars > 0 && h.Repo.StargazersCount < opts.MinStars {
			continue
		}
		if opts.MaxStars > 0 && h.Repo.StargazersCount > opts.MaxStars {
			continue
		}
		if opts.Analyzed != nil {
			if *opts.Analyzed && h.Repo.AnalyzedAt == nil {
				continue
			}
			if !*opts.Analyzed && h.Repo.AnalyzedAt != nil {
				continue
			}
		}
		if opts.AnalysisFailed != nil && *opts.AnalysisFailed && !h.Repo.AnalysisFailed {
			continue
		}
		// Tag filtering: OR logic across ai_tags + topics + custom_tags
		if len(opts.Tags) > 0 {
			match := false
			for _, tag := range opts.Tags {
				tagLower := strings.ToLower(tag)
				for _, t := range h.Repo.AITags {
					if strings.Contains(strings.ToLower(t), tagLower) {
						match = true
						break
					}
				}
				if !match {
					for _, t := range h.Repo.Topics {
						if strings.Contains(strings.ToLower(t), tagLower) {
							match = true
							break
						}
					}
				}
				if !match {
					for _, t := range h.Repo.CustomTags {
						if strings.Contains(strings.ToLower(t), tagLower) {
							match = true
							break
						}
					}
				}
				if match {
					break
				}
			}
			if !match {
				continue
			}
		}
		filtered = append(filtered, h)
	}
	return filtered
}

func buildEmbeddingText(repo *store.Repository, readmeContent string, maxReadmeChars int) string {
	if maxReadmeChars <= 0 {
		maxReadmeChars = 6000
	}
	var parts []string

	if repo.FullName != "" {
		parts = append(parts, fmt.Sprintf("Repository: %s", repo.FullName))
	}

	desc := repo.Description
	summary := repo.AISummary
	if desc != "" && !strings.Contains(summary, desc) {
		parts = append(parts, fmt.Sprintf("Description: %s", desc))
	}
	if repo.CustomDescription != "" {
		parts = append(parts, fmt.Sprintf("About: %s", repo.CustomDescription))
	}
	if summary != "" {
		parts = append(parts, fmt.Sprintf("Summary: %s", summary))
	}

	tagSet := make(map[string]struct{})
	for _, t := range repo.Topics {
		tagSet[t] = struct{}{}
	}
	for _, t := range repo.AITags {
		tagSet[t] = struct{}{}
	}
	for _, t := range repo.CustomTags {
		tagSet[t] = struct{}{}
	}
	allTags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		allTags = append(allTags, t)
	}
	if len(allTags) > 0 {
		parts = append(parts, fmt.Sprintf("Topics: %s", strings.Join(allTags, ", ")))
	}

	if repo.Language != "" {
		parts = append(parts, fmt.Sprintf("Language: %s", repo.Language))
	}

	if readmeContent != "" {
		cleaned := cleanReadme(readmeContent)
		if len(cleaned) > maxReadmeChars {
			cleaned = cleaned[:maxReadmeChars]
		}
		if cleaned != "" {
			parts = append(parts, fmt.Sprintf("README:\n%s", cleaned))
		}
	}

	return strings.Join(parts, "\n")
}

func cleanReadme(content string) string {
	content = regexp.MustCompile(`!\[.*?\]\(.*?\)`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(content, " ")
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	content = regexp.MustCompile(`\[!\[.*?\]\(.*?\)\]\(.*?\)`).ReplaceAllString(content, "")
	return strings.TrimSpace(content)
}
