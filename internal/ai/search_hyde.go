package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Service) generateHyDEQuery(ctx context.Context, userQuery string) (string, error) {
	msgs := []Message{
		{Role: "system", Content: `你是一个 GitHub 仓库推荐专家。用户会描述他们想要的仓库类型，请你生成一段假设的仓库说明文档，用来做语义搜索匹配。

规则：
1. 用 50-100 字的中文描述这个仓库的核心功能、适用场景、技术栈
2. 重点描述功能特性和使用场景，不要说"这是一个..."
3. 保持技术词（如 Go、Rust、React、Kubernetes 等）用英文

输出纯文本，不要 JSON。`},
		{Role: "user", Content: userQuery},
	}
	result, err := s.client.Complete(ctx, msgs)
	if err != nil {
		return userQuery, fmt.Errorf("hyde generation failed: %w", err)
	}
	return strings.TrimSpace(result), nil
}

func hydeEnhance(ctx context.Context, s *Service, query string, enabled bool) string {
	if !enabled || !s.hasAIConfig() {
		return query
	}
	hydeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if hydeResult, err := s.generateHyDEQuery(hydeCtx, query); err == nil && hydeResult != "" {
		return hydeResult
	}
	return query
}
