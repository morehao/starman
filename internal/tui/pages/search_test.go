package pages

import (
	"testing"

	"github.com/morehao/starman/internal/tui/styles"
)

func TestSearchPageWithNilAction(t *testing.T) {
	m := NewSearch(nil, styles.DefaultTheme(), nil)
	m.input.SetValue("test query")
	m.doSearch()
	if m.loaded {
		t.Fatalf("expected not loaded with nil searchAction")
	}
	if m.err != nil {
		t.Fatalf("expected no error with nil searchAction")
	}
}

func TestSearchPageShortQuery(t *testing.T) {
	m := NewSearch(nil, styles.DefaultTheme(), nil)
	m.input.SetValue("t")
	m.doSearch()
	if len(m.results) != 0 {
		t.Fatalf("expected empty results for short query")
	}
}

func TestSearchPageViewNonEmpty(t *testing.T) {
	m := NewSearch(nil, styles.DefaultTheme(), nil)
	v := m.View()
	if v == "" {
		t.Fatalf("expected non-empty view")
	}
}

func TestExtractHitsEmpty(t *testing.T) {
	repos := extractHits(nil)
	if len(repos) != 0 {
		t.Fatalf("expected empty slice from nil hits, got %d", len(repos))
	}
}
