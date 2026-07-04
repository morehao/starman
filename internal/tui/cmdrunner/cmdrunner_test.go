package cmdrunner

import (
	"testing"
)

func TestShellSplit_Simple(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"sync", []string{"sync"}},
		{"sync --full", []string{"sync", "--full"}},
		{`search "go cli" --json`, []string{"search", "go cli", "--json"}},
		{`tag owner/repo +awesome,-old`, []string{"tag", "owner/repo", "+awesome,-old"}},
		{`config show`, []string{"config", "show"}},
		{``, nil},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shellSplit(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("len mismatch: got %v, want %v", got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Fatalf("mismatch at %d: got %q, want %q", i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestShellSplit_QuotedArgs(t *testing.T) {
	input := `search "hello world" --lang "go" --category "dev tools"`
	got := shellSplit(input)
	expected := []string{"search", "hello world", "--lang", "go", "--category", "dev tools"}
	if len(got) != len(expected) {
		t.Fatalf("len mismatch: got %v, want %v", got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Fatalf("mismatch at %d: got %q, want %q", i, got[i], expected[i])
		}
	}
}
