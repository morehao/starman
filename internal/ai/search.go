package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
)

type SearchHit struct {
	Repo  *store.Repository
	Score float64
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
	Language string
	Category string
	Rerank   bool
	Sort     string
	Limit    int
}

func (s *Service) Search(ctx context.Context, query string, st store.Store, opts SearchOpts) ([]*SearchHit, error) {
	intent, err := s.understandQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("understand query: %w", err)
	}

	filters := &store.SearchFilters{
		Language: coalesce(opts.Language, intent.Language),
		Category: coalesce(opts.Category, intent.Category),
		MinStars: intent.MinStars,
		MaxStars: intent.MaxStars,
		Limit:    50,
	}

	ftsResults, err := st.SearchFTS(ctx, intent.FTSQuery, filters)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}

	hits := make([]*SearchHit, 0, len(ftsResults))
	for _, fr := range ftsResults {
		score := calcWeightedScore(fr.BM25Score, fr.Repo.StargazersCount, fr.Repo, intent.Keywords)
		hits = append(hits, &SearchHit{Repo: fr.Repo, Score: score})
	}

	if len(hits) == 0 {
		return hits, nil
	}

	if opts.Rerank && len(hits) > 0 {
		topK := 15
		if topK > len(hits) {
			topK = len(hits)
		}
		hits, err = s.rerank(ctx, query, hits[:topK])
		if err != nil {
			return hits, nil
		}
	}

	sortHits(hits, opts.Sort)

	if opts.Limit > 0 && opts.Limit < len(hits) {
		hits = hits[:opts.Limit]
	}

	return hits, nil
}

func (s *Service) understandQuery(ctx context.Context, query string) (*QueryIntent, error) {
	msgs := []Message{
		{Role: "system", Content: `分析用户的搜索意图，输出 JSON。
{
  "fts_query": "适合 FTS5 全文搜索的查询词，同义词展开，去除停用词",
  "keywords": ["精确匹配关键词"],
  "language": "编程语言（如 Go/Python/JavaScript，如果用户指定了），否则为空",
  "category": "分类名（如 开发工具/AI 机器学习），否则为空",
  "platform": "平台类型（web/desktop/mobile/cli/library/service），否则为空",
  "min_stars": 最低 star 数（整数，默认 0），
  "max_stars": 最高 star 数（整数，默认 0 表示不限）
}`},
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

func calcWeightedScore(bm25 float64, stars int, repo *store.Repository, keywords []string) float64 {
	starNorm := math.Log1p(float64(stars)) / math.Log1p(100000)
	kwScore := keywordMatchScore(repo, keywords)
	return bm25*0.6 + starNorm*0.2 + kwScore*0.2
}

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

type rerankCandidate struct {
	Index      int    `json:"index"`
	FullName   string `json:"full_name"`
	Summary    string `json:"summary"`
	SearchText string `json:"search_text"`
}

type rerankResult struct {
	Rankings []struct {
		Index int     `json:"index"`
		Score float64 `json:"score"`
	} `json:"rankings"`
}

func (s *Service) rerank(ctx context.Context, query string, hits []*SearchHit) ([]*SearchHit, error) {
	candidates := make([]rerankCandidate, len(hits))
	for i, h := range hits {
		candidates[i] = rerankCandidate{
			Index:      i,
			FullName:   h.Repo.FullName,
			Summary:    h.Repo.AISummary,
			SearchText: h.Repo.AISearchText,
		}
	}
	candJSON, _ := json.Marshal(candidates)
	msgs := []Message{
		{Role: "system", Content: fmt.Sprintf(`对候选仓库按查询相关性评分(0-10)，输出 JSON。
查询："%s"
候选仓库列表：
%s
输出格式：{"rankings":[{"index":0,"score":8.5}]}`, query, string(candJSON))},
	}
	resp, err := s.client.Complete(ctx, msgs)
	if err != nil {
		return nil, err
	}
	var result rerankResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("parse rerank: %w", err)
	}
	reranked := make([]*SearchHit, len(result.Rankings))
	for i, r := range result.Rankings {
		if r.Index < len(hits) {
			reranked[i] = hits[r.Index]
			reranked[i].Score = r.Score
		}
	}
	return reranked, nil
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
