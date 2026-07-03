package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
)

type SearchHit struct {
	Repo  *store.Repository
	Score float64
}

type SearchMode string

const (
	SearchModeVector    SearchMode = "vector"
	SearchModeAI        SearchMode = "ai"
	SearchModeBasicText SearchMode = "basic_text"
)

type SearchResult struct {
	Hits []*SearchHit
	Mode SearchMode
}

type QueryIntent struct {
	FTSQuery string   `json:"fts_query"`
	Keywords []string `json:"keywords"`
	Language string   `json:"language"`
	Category string   `json:"category"`
	Platform string   `json:"platform"`
	MinStars int      `json:"min_stars"`
	MaxStars int      `json:"max_stars"`
}

type SearchOpts struct {
	Language       string
	Category       string
	Platform       string
	Tags           []string
	MinStars       int
	MaxStars       int
	Sort           string
	Limit          int
	Analyzed       *bool
	AnalysisFailed *bool
}

func (s *Service) Search(ctx context.Context, query string, st store.Store, opts SearchOpts) (*SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &SearchResult{Mode: SearchModeBasicText, Hits: nil}, nil
	}

	if s.hasEmbeddingConfig() {
		result, err := s.vectorSearch(ctx, query, st, opts)
		if err == nil && len(result.Hits) > 0 {
			result.Mode = SearchModeVector
			return result, nil
		}
		if err != nil {
			log.Printf("vector search failed: %v, falling back", err)
		} else {
			log.Printf("vector search: 0 hits, falling back")
		}
	}

	if s.hasAIConfig() {
		result, err := s.aiSearch(ctx, query, st, opts)
		if err == nil {
			result.Mode = SearchModeAI
			return result, nil
		}
		log.Printf("AI search failed: %v, falling back", err)
	}

	result, err := s.basicTextSearch(ctx, query, st, opts)
	if err != nil {
		return nil, fmt.Errorf("basic text search: %w", err)
	}
	result.Mode = SearchModeBasicText
	return result, nil
}

func (s *Service) hasEmbeddingConfig() bool {
	return s.embeddingClient != nil
}

func (s *Service) hasAIConfig() bool {
	return s.client != nil
}

func (s *Service) aiSearch(
	ctx context.Context, query string, st store.Store, opts SearchOpts,
) (*SearchResult, error) {

	intent, intentErr := s.understandQuery(ctx, query)
	if intentErr != nil {
		intent = &QueryIntent{}
	}
	if intent.FTSQuery == "" {
		intent.FTSQuery = query
	}

	if err := s.searchIndex.ensureLoaded(ctx, st); err != nil {
		return nil, fmt.Errorf("load search index: %w", err)
	}

	filters := buildSearchFilters(opts, intent)
	hits := s.searchIndex.Search(intent.FTSQuery, intent.Keywords, filters, 50)

	if len(hits) == 0 {
		return &SearchResult{Hits: []*SearchHit{}}, nil
	}

	sortHits(hits, opts.Sort)
	if opts.Limit > 0 && opts.Limit < len(hits) {
		hits = hits[:opts.Limit]
	}

	return &SearchResult{Hits: hits}, nil
}

func (s *Service) understandQuery(ctx context.Context, query string) (*QueryIntent, error) {
	msgs := []Message{
		{Role: "system", Content: `你是一个搜索查询分析器，分析用户的搜索意图后输出 JSON。
{
  "fts_query": "中英文混合搜索词，保留中文原词并添加英文同义词，用空格分隔",
  "keywords": ["中文关键词", "英文关键词", "同义词"],
  "language": "编程语言（如 Go/Python/JavaScript，如果用户指定了），否则为空",
  "category": "分类名（如 开发工具/AI 机器学习），否则为空",
  "platform": "平台类型（web/desktop/mobile/cli/library/service），否则为空",
  "min_stars": 最低 star 数（整数，默认 0），
  "max_stars": 最高 star 数（整数，默认 0 表示不限）
}

示例：用户输入"好看的终端工具"
输出：{"fts_query": "好看的 终端 工具 terminal cli command-line", "keywords": ["终端", "工具", "terminal", "cli", "command-line"], "language": "", "category": "", "platform": "cli", "min_stars": 0, "max_stars": 0}

fts_query 和 keywords 都需要同时包含中文原文和英文翻译，提高搜索召回率。`},
		{Role: "user", Content: query},
	}
	resp, err := s.client.Complete(ctx, msgs)
	if err != nil {
		return nil, err
	}
	var intent QueryIntent
	if err := json.Unmarshal([]byte(resp), &intent); err != nil {
		return nil, fmt.Errorf("parse query intent: %w", err)
	}
	if intent.FTSQuery == "" {
		intent.FTSQuery = query
	}
	return &intent, nil
}

// Deprecated: calcWeightedScore was used for FTS5 BM25-based search scoring.
// Scoring is now handled by SearchIndex.scoreRepo in search_index.go.
func calcWeightedScore(bm25 float64, stars int, repo *store.Repository, keywords []string) float64 {
	starNorm := math.Log1p(float64(stars)) / math.Log1p(100000)
	kwScore := keywordMatchScore(repo, keywords)
	return bm25*0.6 + starNorm*0.2 + kwScore*0.2
}

// Deprecated: keywordMatchScore was used for FTS5 BM25-based search scoring.
// Keyword matching is now handled by SearchIndex.scoreRepo in search_index.go.
func keywordMatchScore(repo *store.Repository, keywords []string) float64 {
	var score float64
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if containsFold(repo.FullName, kwLower) {
			score += 3
		}
		if containsFold(repo.AISearchText, kwLower) {
			score += 2
		}
		if containsFold(repo.Description, kwLower) {
			score += 1
		}
	}
	return score
}

func sortHits(hits []*SearchHit, sortBy string) {
	switch sortBy {
	case "stars":
		sort.Slice(hits, func(i, j int) bool {
			return hits[i].Repo.StargazersCount > hits[j].Repo.StargazersCount
		})
	case "name":
		sort.Slice(hits, func(i, j int) bool {
			return hits[i].Repo.FullName < hits[j].Repo.FullName
		})
	default:
		sort.Slice(hits, func(i, j int) bool {
			return hits[i].Score > hits[j].Score
		})
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
