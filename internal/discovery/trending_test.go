package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>GitHub Trending</title>
    <item>
      <title>owner1/repo1</title>
      <link>https://github.com/owner1/repo1</link>
      <description>⭐ 1,234 | 🍴 56 | A CLI tool for managing stars</description>
    </item>
    <item>
      <title>owner2/repo2</title>
      <link>https://github.com/owner2/repo2</link>
      <description>⭐ 5,678 | 🍴 90 | A web framework</description>
    </item>
  </channel>
</rss>`

func TestParseRSS(t *testing.T) {
	repos, err := parseRSS([]byte(sampleRSS))
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].FullName != "owner1/repo1" {
		t.Fatalf("expected owner1/repo1, got %s", repos[0].FullName)
	}
	if repos[0].Stars != 1234 {
		t.Fatalf("expected 1234 stars, got %d", repos[0].Stars)
	}
	if repos[0].Forks != 56 {
		t.Fatalf("expected 56 forks, got %d", repos[0].Forks)
	}
	if repos[1].Stars != 5678 {
		t.Fatalf("expected 5678 stars, got %d", repos[1].Stars)
	}
}

func TestParseRSSEmpty(t *testing.T) {
	repos, err := parseRSS([]byte(`<?xml version="1.0"?><rss><channel></channel></rss>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 0 {
		t.Fatalf("expected 0 repos, got %d", len(repos))
	}
}

func TestTrendingViaRSS(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()
	svc := &Service{rssURL: srv.URL, http: &http.Client{}}
	ctx := context.Background()
	repos, err := svc.trendingViaRSS(ctx, TrendingOpts{Since: "weekly", Top: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].Rank != 1 {
		t.Fatalf("expected rank 1, got %d", repos[0].Rank)
	}
}

func TestTrendingRSSFallbackOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", 500)
	}))
	defer srv.Close()
	svc := &Service{rssURL: srv.URL, http: &http.Client{}}
	ctx := context.Background()
	_, err := svc.trendingViaRSS(ctx, TrendingOpts{Since: "weekly", Top: 10})
	if err == nil {
		t.Fatal("expected error on RSS failure")
	}
}
