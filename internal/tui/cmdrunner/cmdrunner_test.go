package cmdrunner

import (
	"context"
	"errors"
	"strings"
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

func TestRunner_Run_SyncHelp(t *testing.T) {
	r := NewRunner("test")
	stdout, stderr, err := r.Run(context.Background(), "sync --help")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Sync starred") {
		t.Errorf("expected sync help in stdout, got: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got: %s", stderr)
	}
}

func TestRunner_Run_ConfigInitInteractive(t *testing.T) {
	r := NewRunner("test")
	_, _, err := r.Run(context.Background(), "config init")
	if !errors.Is(err, ErrInteractiveRequired) {
		t.Errorf("expected ErrInteractiveRequired, got: %v", err)
	}
}

func TestRunner_Run_SyncWatchBlocking(t *testing.T) {
	r := NewRunner("test")
	_, _, err := r.Run(context.Background(), "sync --watch")
	if !errors.Is(err, ErrBlockingRequired) {
		t.Errorf("expected ErrBlockingRequired, got: %v", err)
	}
}

func TestRunner_Run_InvalidCommand(t *testing.T) {
	r := NewRunner("test")
	_, _, err := r.Run(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
}

func TestRunner_Run_EmptyInput(t *testing.T) {
	r := NewRunner("test")
	stdout, stderr, err := r.Run(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "" {
		t.Errorf("expected empty stdout, got: %s", stdout)
	}
	if stderr != "" {
		t.Errorf("expected empty stderr, got: %s", stderr)
	}
}
