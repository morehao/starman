package styles

import "testing"

func TestDefaultTheme(t *testing.T) {
	th := DefaultTheme()
	if th == nil {
		t.Fatal("expected non-nil theme")
	}
	if th.Sidebar.GetWidth() != 30 {
		t.Errorf("sidebar width = %d, want 30", th.Sidebar.GetWidth())
	}
}
