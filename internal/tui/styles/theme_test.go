package styles

import "testing"

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th == nil {
		t.Fatal("expected non-nil theme")
	}
	if th.Primary == "" {
		t.Error("expected non-empty primary color")
	}
}
