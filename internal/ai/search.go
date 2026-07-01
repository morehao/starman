package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/morehao/starman/internal/store"
)

type SearchHit struct {
	Repo  *store.Repository
	Score float64
}

func (s *Service) Search(ctx context.Context, query string, repos []*store.Repository) ([]*SearchHit, error) {
	keywords, err := s.translateQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("translate query: %w", err)
	}
	hits := make([]*SearchHit, 0, len(repos))
	for _, r := range repos {
		score := scoreRepo(r, keywords)
		if score > 0 {
			hits = append(hits, &SearchHit{Repo: r, Score: score})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	return hits, nil
}

func (s *Service) translateQuery(ctx context.Context, query string) ([]string, error) {
	msgs := []Message{
		{Role: "system", Content: `将用户的搜索意图翻译为3-5个英文关键词，输出JSON：{"keywords":["word1","word2"]}`},
		{Role: "user", Content: query},
	}
	resp, err := s.client.Complete(ctx, msgs)
	if err != nil {
		return nil, err
	}
	var result struct {
		Keywords []string `json:"keywords"`
	}
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("parse keywords: %w", err)
	}
	return result.Keywords, nil
}

func scoreRepo(r *store.Repository, keywords []string) float64 {
	var score float64
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if containsFold(r.FullName, kwLower) {
			score += 3
		}
		if r.Description != "" && containsFold(r.Description, kwLower) {
			score += 2
		}
		if r.AISummary != "" && containsFold(r.AISummary, kwLower) {
			score += 2
		}
		for _, tag := range r.AITags {
			if containsFold(tag, kwLower) {
				score += 3
				break
			}
		}
		for _, topic := range r.Topics {
			if containsFold(topic, kwLower) {
				score += 1
				break
			}
		}
		if r.Language != "" && containsFold(r.Language, kwLower) {
			score += 1
		}
	}
	return score
}
