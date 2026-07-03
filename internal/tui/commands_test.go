package tui

import (
	"testing"
)

func TestParseCommandEmpty(t *testing.T) {
	cmd := parseCommand("")
	if cmd.Name != "" {
		t.Fatalf("expected empty name, got %q", cmd.Name)
	}
}

func TestParseCommandSimple(t *testing.T) {
	cmd := parseCommand("sync")
	if cmd.Name != "sync" {
		t.Fatalf("expected sync, got %q", cmd.Name)
	}
}

func TestParseCommandWithArgs(t *testing.T) {
	cmd := parseCommand("sync --full --limit=50")
	if cmd.Name != "sync" {
		t.Fatalf("expected sync, got %q", cmd.Name)
	}
	if cmd.Flags["full"] != "true" {
		t.Fatalf("expected full=true, got %q", cmd.Flags["full"])
	}
	if cmd.Flags["limit"] != "50" {
		t.Fatalf("expected limit=50, got %q", cmd.Flags["limit"])
	}
}

func TestParseCommandWithPositionalArgs(t *testing.T) {
	cmd := parseCommand("tag owner/repo +awesome")
	if cmd.Name != "tag" {
		t.Fatalf("expected tag, got %q", cmd.Name)
	}
	if len(cmd.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(cmd.Args))
	}
	if cmd.Args[0] != "owner/repo" {
		t.Fatalf("expected owner/repo, got %q", cmd.Args[0])
	}
	if cmd.Args[1] != "+awesome" {
		t.Fatalf("expected +awesome, got %q", cmd.Args[1])
	}
}

func TestParseCommandQuit(t *testing.T) {
	for _, q := range []string{"q", "quit"} {
		cmd := parseCommand(q)
		if cmd.Name != q {
			t.Fatalf("expected %q, got %q", q, cmd.Name)
		}
	}
}
