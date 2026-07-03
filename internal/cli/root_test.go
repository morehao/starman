package cli

import "testing"

func TestRootCommandSet(t *testing.T) {
	root := NewRootCmd("test")
	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}
	wantPresent := []string{"sync", "search", "config", "completion", "help"}
	for _, name := range wantPresent {
		if !got[name] {
			t.Fatalf("expected command %q to exist", name)
		}
	}
	wantAbsent := []string{"analyze", "tag", "categorize", "stats", "release", "generate", "backup", "info", "trending"}
	for _, name := range wantAbsent {
		if got[name] {
			t.Fatalf("expected command %q to be removed from root", name)
		}
	}
}
