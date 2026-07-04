package trendingsection

import (
	"testing"

	"github.com/morehao/starman/internal/discovery"
)

func TestTrendingRowColumns(t *testing.T) {
	row := TrendingRow{
		Repo: &discovery.TrendingRepo{
			FullName:    "owner/repo",
			Stars:       1000,
			Language:    "Go",
			Description: "A great tool",
		},
	}

	if row.GetTitle() != "owner/repo" {
		t.Fatalf("unexpected title: %s", row.GetTitle())
	}

	cols := row.GetColumns()
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(cols))
	}
	if cols[0] != "owner/repo" {
		t.Fatalf("unexpected col 0: %s", cols[0])
	}
	if cols[2] != "Go" {
		t.Fatalf("unexpected col 2: %s", cols[2])
	}
}

func TestTrendingRowEmptyFields(t *testing.T) {
	row := TrendingRow{
		Repo: &discovery.TrendingRepo{
			FullName: "owner/repo",
		},
	}
	cols := row.GetColumns()
	if cols[2] != "-" {
		t.Fatalf("expected - for empty language, got %s", cols[2])
	}
}

func TestTrendingToStoreRepo(t *testing.T) {
	tr := &discovery.TrendingRepo{
		FullName:        "owner/repo",
		Description:     "desc",
		URL:             "https://example.com",
		Language:        "Go",
		Stars:           500,
		Forks:           10,
		Topics:          []string{"tui"},
	}
	sr := TrendingToStoreRepo(tr)
	if sr.FullName != "owner/repo" {
		t.Fatalf("unexpected fullname: %s", sr.FullName)
	}
	if sr.StargazersCount != 500 {
		t.Fatalf("unexpected stars: %d", sr.StargazersCount)
	}
	if len(sr.Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(sr.Topics))
	}
}

func TestFormatStarCount(t *testing.T) {
	if formatStarCount(128) != "128" {
		t.Fatal("expected 128")
	}
	if formatStarCount(3200) != "3.2k" {
		t.Fatalf("expected 3.2k, got %s", formatStarCount(3200))
	}
}

func TestStringPeriod(t *testing.T) {
	if StringPeriod("daily") != "Daily" {
		t.Fatalf("expected Daily, got %s", StringPeriod("daily"))
	}
	if StringPeriod("weekly") != "Weekly" {
		t.Fatalf("expected Weekly, got %s", StringPeriod("weekly"))
	}
}
