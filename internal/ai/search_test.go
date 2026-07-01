package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(completionResponse{
			Choices: []struct {
				Message Message `json:"message"`
			}{{Message: Message{Content: `{"keywords":["terminal","cli"]}`}}},
		})
	}))
	defer server.Close()
	c := NewClient(server.URL, "key", "model")
	svc := NewService(c, nil)
	repos := []*store.Repository{
		{ID: 1, FullName: "owner/cli-tool", Description: "A terminal tool", AITags: []string{"cli"}, Topics: []string{"go"}},
		{ID: 2, FullName: "owner/cooking-app", Description: "Recipe manager", AITags: []string{"food"}},
	}
	hits, err := svc.Search(context.Background(), "终端工具", repos)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Repo.ID != 1 {
		t.Fatalf("expected repo 1, got %d", hits[0].Repo.ID)
	}
	if hits[0].Score <= 0 {
		t.Fatal("expected positive score")
	}
}

func TestScoreRepo(t *testing.T) {
	r := &store.Repository{
		FullName:    "owner/awesome-cli",
		Description: "A CLI tool",
		AITags:      []string{"cli", "tool"},
		Topics:      []string{"go", "cli"},
		Language:    "Go",
	}
	score := scoreRepo(r, []string{"cli"})
	if score < 6 {
		t.Fatalf("expected score >= 6 (fullname+tags+topics), got %f", score)
	}
}
