package cli

import (
	"testing"

	"github.com/morehao/starman/internal/ai"
)

func TestSearchCommandUsesAction(t *testing.T) {
	cmd := newSearchCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Flags().Lookup("limit") == nil {
		t.Fatal("expected --limit flag")
	}
	if cmd.Flags().Lookup("lang") == nil {
		t.Fatal("expected --lang flag")
	}
	if cmd.Flags().Lookup("category") == nil {
		t.Fatal("expected --category flag")
	}
	if cmd.Flags().Lookup("sort") == nil {
		t.Fatal("expected --sort flag")
	}
	if cmd.Flags().Lookup("rerank") != nil {
		t.Fatal("--rerank flag should not exist")
	}
}

func TestSearchCmd_NoRerankFlag(t *testing.T) {
	cmd := newSearchCmd()
	if cmd.Flags().Lookup("rerank") != nil {
		t.Fatal("--rerank flag was not removed")
	}
}

func TestOutputSearchJSON_NilHits(t *testing.T) {
	err := outputSearchJSON(nil)
	if err != nil {
		t.Fatalf("expected no error for nil hits, got %v", err)
	}
}

func TestOutputSearchJSON_EmptyHits(t *testing.T) {
	err := outputSearchJSON([]*ai.SearchHit{})
	if err != nil {
		t.Fatalf("expected no error for empty hits, got %v", err)
	}
}
