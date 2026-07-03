package footer

import (
	"strings"
	"testing"
)

func TestFooterViewContainsHelpHint(t *testing.T) {
	f := NewModel(nil)
	out := f.View()
	if !strings.Contains(out, "?help") {
		t.Fatalf("missing help hint: %q", out)
	}
}

func TestFooterViewContainsLeftAndRight(t *testing.T) {
	f := NewModel(nil)
	f.SetLeft("left")
	f.SetRight("right")
	out := f.View()
	if !strings.Contains(out, "left") {
		t.Fatalf("missing left value: %q", out)
	}
	if !strings.Contains(out, "right") {
		t.Fatalf("missing right value: %q", out)
	}
}
