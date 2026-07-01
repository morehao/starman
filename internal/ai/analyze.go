package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/morehao/starman/internal/github"
	"github.com/morehao/starman/internal/store"
)

type AnalysisResult struct {
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	Platforms  []string `json:"platforms"`
	SearchText string   `json:"search_text"`
}

type Service struct {
	client *Client
	gh     *github.Client
}

func NewService(client *Client, gh *github.Client) *Service {
	return &Service{client: client, gh: gh}
}

const readmeMaxChars = 8000

func (s *Service) AnalyzeRepository(ctx context.Context, repo *store.Repository, readme string, cats []*store.Category) (*AnalysisResult, error) {
	readme = truncate(readme, readmeMaxChars)
	msgs := buildAnalyzeMessages(repo, readme, cats)
	resp, err := s.client.Complete(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("ai complete: %w", err)
	}
	var result AnalysisResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("parse ai response: %w (raw: %s)", err, resp)
	}
	return &result, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func buildAnalyzeMessages(repo *store.Repository, readme string, cats []*store.Category) []Message {
	catNames := make([]string, 0, len(cats))
	for _, c := range cats {
		if !c.IsHidden {
			catNames = append(catNames, c.Name)
		}
	}
	system := fmt.Sprintf(`你是一个 GitHub 仓库分析助手。根据仓库信息和 README，输出 JSON：
{"summary": "一句话摘要(中文,≤80字)", "tags": ["3-5个标签"], "platforms": ["web|desktop|mobile|cli|library|service"], "search_text": "扩展检索描述"}

search_text 用中文描述仓库的核心功能、适用场景、技术栈，50-150字，用于全文搜索匹配。

可选分类（tags 尽量从中选取，也可补充）：%s`, strings.Join(catNames, "、"))

	user := fmt.Sprintf(`仓库：%s
语言：%s
描述：%s
Topics：%s
README：%s`,
		repo.FullName,
		repo.Language,
		repo.Description,
		strings.Join(repo.Topics, ", "),
		readme,
	)
	return []Message{
		{Role: "system", Content: system},
		{Role: "user", Content: user},
	}
}
