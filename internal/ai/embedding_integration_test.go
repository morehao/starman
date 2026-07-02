//go:build integration

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/morehao/starman/internal/config"
)

func resolveEmbeddingEnv(t *testing.T) (baseURL, apiKey, model string) {
	t.Helper()

	apiKey = os.Getenv("STARMAN_EMBEDDING_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("STARMAN_AI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("STARMAN_EMBEDDING_API_KEY or STARMAN_AI_API_KEY not set, skipping embedding test")
	}

	baseURL = os.Getenv("STARMAN_EMBEDDING_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("STARMAN_AI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model = os.Getenv("STARMAN_EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}
	return
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func callEmbeddingAPI(ctx context.Context, baseURL, apiKey, model, input string) (*embeddingResponse, error) {
	body := embeddingRequest{Model: model, Input: input}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/embeddings", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("embedding endpoint not found (status 404): %s — provider may not support embeddings", string(respBody))
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embedding api error %d: %s", resp.StatusCode, string(respBody))
	}

	var result embeddingResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return &result, nil
}

func TestEmbeddingAPI_Call(t *testing.T) {
	baseURL, apiKey, model := resolveEmbeddingEnv(t)

	ctx := context.Background()

	result, err := callEmbeddingAPI(ctx, baseURL, apiKey, model, "hello world")
	if err != nil {
		if strings.Contains(err.Error(), "not found (status 404)") {
			t.Skipf("embedding endpoint not available at %s, skipping", baseURL)
		}
		t.Fatalf("embedding API call failed: %v", err)
	}

	if len(result.Data) == 0 {
		t.Fatal("expected at least 1 embedding vector")
	}
	if len(result.Data[0].Embedding) == 0 {
		t.Fatal("expected non-empty embedding vector")
	}
	if result.Usage.TotalTokens == 0 {
		t.Error("expected non-zero token usage")
	}

	t.Logf("embedding: model=%s, dimensions=%d, tokens=%d",
		model, len(result.Data[0].Embedding), result.Usage.TotalTokens)
}

func TestEmbeddingAPI_Batch(t *testing.T) {
	baseURL, apiKey, model := resolveEmbeddingEnv(t)

	ctx := context.Background()

	inputs := []string{
		"A command-line tool for managing GitHub stars",
		"A web framework for building REST APIs",
		"一款管理 GitHub 星标仓库的终端工具",
	}

	for _, input := range inputs {
		result, err := callEmbeddingAPI(ctx, baseURL, apiKey, model, input)
		if err != nil {
			if strings.Contains(err.Error(), "not found (status 404)") {
				t.Skipf("embedding endpoint not available at %s, skipping", baseURL)
			}
			t.Fatalf("embedding for '%s' failed: %v", input, err)
		}
		dim := len(result.Data[0].Embedding)
		t.Logf("input=%q → dims=%d tokens=%d", input, dim, result.Usage.TotalTokens)
	}
}

func TestResolveEmbeddingKey(t *testing.T) {
	t.Run("flag token priority", func(t *testing.T) {
		cfg := config.Default()
		key := config.ResolveEmbeddingKey(cfg, "flag-key")
		if key != "flag-key" {
			t.Errorf("expected flag-key, got %s", key)
		}
	})

	t.Run("env var fallback", func(t *testing.T) {
		t.Setenv("STARMAN_EMBEDDING_API_KEY", "env-embed-key")
		cfg := config.Default()
		key := config.ResolveEmbeddingKey(cfg, "")
		if key != "env-embed-key" {
			t.Errorf("expected env-embed-key, got %s", key)
		}
	})

	t.Run("config fallback", func(t *testing.T) {
		t.Setenv("STARMAN_EMBEDDING_API_KEY", "")
		cfg := config.Default()
		cfg.Embedding.APIKey = "cfg-embed-key"
		key := config.ResolveEmbeddingKey(cfg, "")
		if key != "cfg-embed-key" {
			t.Errorf("expected cfg-embed-key, got %s", key)
		}
	})

	t.Run("empty when nothing set", func(t *testing.T) {
		t.Setenv("STARMAN_EMBEDDING_API_KEY", "")
		cfg := config.Default()
		key := config.ResolveEmbeddingKey(cfg, "")
		if key != "" {
			t.Errorf("expected empty, got %s", key)
		}
	})
}
