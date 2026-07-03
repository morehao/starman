package pages

import (
	"testing"

	"github.com/morehao/starman/internal/tui/styles"
)

func TestTrendingPageNilDiscovery(t *testing.T) {
	m := NewTrending(nil, styles.DefaultTheme(), nil)
	msg := m.loadCmd()
	loaded, ok := msg.(trendingLoadedMsg)
	if !ok {
		t.Fatalf("expected trendingLoadedMsg from loadCmd, got %T", msg)
	}
	if loaded.repos != nil {
		t.Fatalf("expected nil repos with nil discovery")
	}
}

func TestTrendingPageViewLoading(t *testing.T) {
	m := NewTrending(nil, styles.DefaultTheme(), nil)
	v := m.View()
	if v != "loading trending..." {
		t.Fatalf("expected loading message, got %q", v)
	}
}

func TestTrendingPageTabSwitchesSource(t *testing.T) {
	m := NewTrending(nil, styles.DefaultTheme(), nil)
	if m.source != "rss" {
		t.Fatalf("expected initial source rss, got %s", m.source)
	}
}

func TestTrendingPageInit(t *testing.T) {
	m := NewTrending(nil, styles.DefaultTheme(), nil)
	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected non-nil init command")
	}
}
