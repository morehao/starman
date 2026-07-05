package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type EmbeddingClient struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
	dim     atomic.Int32
}

func NewEmbeddingClient(baseURL, apiKey, model string) *EmbeddingClient {
	if apiKey == "" || baseURL == "" {
		return nil
	}
	return &EmbeddingClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type embeddingRequest struct {
	Input          any    `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (ec *EmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if ec == nil {
		return nil, fmt.Errorf("embedding client not configured")
	}
	body := embeddingRequest{
		Input:          texts,
		Model:          ec.model,
		EncodingFormat: "float",
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", ec.baseURL+"/embeddings", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ec.apiKey)
	resp, err := ec.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		return nil, fmt.Errorf("embedding API error %d: %s", resp.StatusCode, errBody.String())
	}
	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	vectors := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		vectors[d.Index] = d.Embedding
	}
	if len(vectors) > 0 && len(vectors[0]) > 0 && ec.dim.Load() == 0 {
		ec.dim.Store(int32(len(vectors[0])))
	}
	return vectors, nil
}

func (ec *EmbeddingClient) Dimension() int {
	if ec == nil {
		return 0
	}
	return int(ec.dim.Load())
}
