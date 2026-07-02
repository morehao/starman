package styles

import "testing"

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th == nil {
		t.Fatal("expected non-nil theme")
	}
	if th.Sidebar.GetWidth() != 24 {
		t.Errorf("sidebar width = %d, want 24", th.Sidebar.GetWidth())
	}
}
