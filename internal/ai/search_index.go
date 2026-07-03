package ai

import (
	"context"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/morehao/starman/internal/store"
)

type repoEntry struct {
	repo       *store.Repository
	searchText string
}

type SearchIndex struct {
	mu      sync.RWMutex
	entries []*repoEntry
	loaded  bool
}

func NewSearchIndex() *SearchIndex {
	return &SearchIndex{}
}

func (idx *SearchIndex) Load(ctx context.Context, st store.Store) error {
	repos, err := st.ListRepositories(ctx)
	if err != nil {
		return err
	}

	entries := make([]*repoEntry, len(repos))
	for i, r := range repos {
		entries[i] = &repoEntry{repo: r, searchText: buildSearchText(r)}
	}

	idx.mu.Lock()
	idx.entries = entries
	idx.loaded = true
	idx.mu.Unlock()
	return nil
}

func (idx *SearchIndex) ensureLoaded(ctx context.Context, st store.Store) error {
	idx.mu.RLock()
	if idx.loaded {
		idx.mu.RUnlock()
		return nil
	}
	idx.mu.RUnlock()
	return idx.Load(ctx, st)
}

func (idx *SearchIndex) Search(query string, keywords []string, filters *store.SearchFilters, limit int) []*SearchHit {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	queryLower := strings.ToLower(strings.TrimSpace(query))
	var hits []*SearchHit

	for _, entry := range idx.entries {
		if !matchRepo(entry.repo, filters) {
			continue
		}

		score := scoreRepo(entry, queryLower, keywords)
		if score > 0 || queryLower == "" {
			hits = append(hits, &SearchHit{Repo: entry.repo, Score: score})
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })

	if limit > 0 && limit < len(hits) {
		hits = hits[:limit]
	}

	return hits
}

func matchRepo(repo *store.Repository, filters *store.SearchFilters) bool {
	if filters == nil {
		return true
	}
	if filters.Language != "" && !strings.EqualFold(repo.Language, filters.Language) {
		return false
	}
	if filters.Category != "" {
		cat := repo.CustomCategory
		if cat == "" {
			cat = repo.AICategory
		}
		if !strings.EqualFold(cat, filters.Category) {
			return false
		}
	}
	if filters.MinStars > 0 && repo.StargazersCount < filters.MinStars {
		return false
	}
	if filters.MaxStars > 0 && repo.StargazersCount > filters.MaxStars {
		return false
	}
	if filters.Platform != "" {
		platMatch := false
		for _, p := range repo.AIPlatforms {
			if strings.EqualFold(p, filters.Platform) {
				platMatch = true
				break
			}
		}
		if !platMatch {
			return false
		}
	}
	if len(filters.Tags) > 0 {
		match := false
		for _, tag := range filters.Tags {
			tagLower := strings.ToLower(tag)
			for _, t := range repo.AITags {
				if strings.Contains(strings.ToLower(t), tagLower) {
					match = true
					break
				}
			}
			if !match {
				for _, t := range repo.Topics {
					if strings.Contains(strings.ToLower(t), tagLower) {
						match = true
						break
					}
				}
			}
			if !match {
				for _, t := range repo.CustomTags {
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
			return false
		}
	}
	if filters.Analyzed != nil {
		if *filters.Analyzed && (repo.AnalyzedAt == nil || repo.AnalysisFailed) {
			return false
		}
		if !*filters.Analyzed && repo.AnalyzedAt != nil {
			return false
		}
	}
	if filters.AnalysisFailed != nil && *filters.AnalysisFailed && !repo.AnalysisFailed {
		return false
	}
	return true
}

func scoreRepo(entry *repoEntry, query string, keywords []string) float64 {
	var score float64

	if query != "" {
		if strings.Contains(entry.searchText, query) {
			score += 10.0
		}
		for _, part := range strings.Fields(query) {
			if len(part) >= 2 && strings.Contains(entry.searchText, part) {
				score += 3.0
			}
		}
	}

	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		fnLower := strings.ToLower(entry.repo.FullName)
		if strings.Contains(fnLower, kwLower) {
			score += 3.0
		}
		score += float64(strings.Count(entry.searchText, kwLower)) * 0.5
	}

	starNorm := math.Log1p(float64(entry.repo.StargazersCount)) / math.Log1p(100000)
	score += starNorm * 2.0

	return score / (score + 5.0)
}

func buildSearchText(repo *store.Repository) string {
	var sb strings.Builder
	write := func(s string) {
		if s != "" {
			sb.WriteString(strings.ToLower(s))
			sb.WriteByte(' ')
		}
	}
	write(repo.Name)
	write(repo.FullName)
	write(repo.Description)
	write(repo.CustomDescription)
	write(repo.AISearchText)
	write(repo.AISummary)
	write(repo.AICategory)
	write(repo.Language)
	for _, t := range repo.Topics {
		write(t)
	}
	for _, t := range repo.AITags {
		write(t)
	}
	for _, t := range repo.CustomTags {
		write(t)
	}
	return sb.String()
}
