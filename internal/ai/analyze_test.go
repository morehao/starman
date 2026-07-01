package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestAnalyzeRepository(t *testing.T) {
	aiResp := AnalysisResult{Summary: "一个好用的工具", Tags: []string{"cli", "go"}, Platforms: []string{"cli"}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(completionResponse{
			Choices: []struct {
				Message Message `json:"message"`
			}{{Message: Message{Content: mustJSON(aiResp)}}},
		})
	}))
	defer server.Close()
	c := NewClient(server.URL, "key", "model")
	svc := NewService(c, nil)
	repo := &store.Repository{FullName: "owner/repo", Language: "Go", Description: "A tool", Topics: []string{"go"}}
	cats := []*store.Category{{ID: "dev-tools", Name: "开发工具", Keywords: []string{"cli"}}}
	result, err := svc.AnalyzeRepository(context.Background(), repo, "# README", cats)
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary != "一个好用的工具" {
		t.Fatalf("unexpected summary: %s", result.Summary)
	}
	if len(result.Tags) != 2 || result.Tags[0] != "cli" {
		t.Fatalf("unexpected tags: %v", result.Tags)
	}
}

func TestTruncate(t *testing.T) {
	s := "hello world"
	if got := truncate(s, 100); got != s {
		t.Fatalf("expected unchanged, got %s", got)
	}
	if got := truncate(s, 5); got != "hello" {
		t.Fatalf("expected hello, got %s", got)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
