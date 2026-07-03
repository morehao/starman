package tabs

import (
	"strings"
	"testing"
)

func TestTabsViewContainsSectionTitles(t *testing.T) {
	m := NewModel(nil)
	m.SetTitles([]string{"Stars", "Trending", "Releases"})
	out := m.View()
	if !strings.Contains(out, "Stars") {
		t.Fatalf("missing Stars title: %q", out)
	}
	if !strings.Contains(out, "Trending") {
		t.Fatalf("missing Trending title: %q", out)
	}
	if !strings.Contains(out, "Releases") {
		t.Fatalf("missing Releases title: %q", out)
	}
}
