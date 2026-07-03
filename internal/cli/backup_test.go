package cli

import (
	"testing"
)

func TestBackupRepoFlags(t *testing.T) {
	cmd := newBackupCmd()
	repoFlag := cmd.Flags().Lookup("repo")
	if repoFlag == nil {
		t.Fatal("expected --repo flag")
	}
	if repoFlag.DefValue != "" {
		t.Fatalf("expected default empty, got %q", repoFlag.DefValue)
	}
}

func TestBackupMessageFlag(t *testing.T) {
	cmd := newBackupCmd()
	msgFlag := cmd.Flags().Lookup("message")
	if msgFlag == nil {
		t.Fatal("expected --message flag")
	}
	if msgFlag.DefValue != "" {
		t.Fatalf("expected default empty, got %q", msgFlag.DefValue)
	}
}
