package inputoverlay

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/morehao/starman/internal/tui/theme"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

func goldenPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("testdata", t.Name()+".golden")
}

func updateGolden() bool {
	return os.Getenv("UPDATE_GOLDEN") == "1"
}

func TestRenderOverlay_ZeroSize(t *testing.T) {
	th := theme.DefaultTheme()
	got := RenderOverlay(th, 0, 0, "Search", "test")
	if got == "" {
		t.Error("expected non-empty render even with zero dimensions")
	}
}

func TestRenderOverlay_NarrowWidth(t *testing.T) {
	th := theme.DefaultTheme()
	tests := []struct {
		name  string
		width int
	}{
		{"very_narrow", 20},
		{"clamped_width", 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderOverlay(th, tt.width, 10, "Search", "test")
			if got == "" {
				t.Error("expected non-empty render")
			}
		})
	}
}

func TestRenderOverlay_EmptyQuery(t *testing.T) {
	th := theme.DefaultTheme()
	got := RenderOverlay(th, 80, 24, "Search", "")
	if got == "" {
		t.Error("expected non-empty render with empty query")
	}
}

func TestRenderOverlay_NormalCase(t *testing.T) {
	th := theme.DefaultTheme()
	got := RenderOverlay(th, 80, 24, "Search", "hello world")
	if got == "" {
		t.Error("expected non-empty render")
	}
}

func TestRenderOverlay_ContentWidthNegative(t *testing.T) {
	th := theme.DefaultTheme()
	got := RenderOverlay(th, 8, 10, "Search", "test")
	if got == "" {
		t.Error("expected non-empty render with very narrow width")
	}
}

func TestRenderOverlay_Golden(t *testing.T) {
	th := theme.DefaultTheme()
	tests := []struct {
		name  string
		width int
		query string
	}{
		{"focused_empty", 80, ""},
		{"focused_query", 80, "go cli"},
		{"narrow_width", 50, "go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(RenderOverlay(th, tt.width, 24, "Search", tt.query))

			path := goldenPath(t)
			if updateGolden() {
				err := os.MkdirAll(filepath.Dir(path), 0o755)
				if err != nil {
					t.Fatalf("failed to create testdata dir: %v", err)
				}
				err = os.WriteFile(path, []byte(got), 0o644)
				if err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read golden file: %v (run with UPDATE_GOLDEN=1 to generate)", err)
			}
			if string(want) != got {
				t.Errorf("golden mismatch:\n--- want:\n%s\n--- got:\n%s", string(want), got)
			}
		})
	}
}
