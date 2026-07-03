package releasessection

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestReleaseRowColumns(t *testing.T) {
	row := ReleaseRow{
		Release: &store.Release{
			RepoFullName: "owner/repo",
			TagName:      "v1.0.0",
			PublishedAt:  "2025-06-01T00:00:00Z",
			IsRead:       false,
		},
	}

	cols := row.GetColumns()
	if len(cols) != 4 {
		t.Fatalf("expected 4 columns, got %d", len(cols))
	}
	if cols[0] != "owner/repo" {
		t.Fatalf("unexpected col 0: %s", cols[0])
	}
	if cols[1] != "v1.0.0" {
		t.Fatalf("unexpected col 1: %s", cols[1])
	}
	if cols[2] != "2025-06-01" {
		t.Fatalf("unexpected col 2: %s", cols[2])
	}
	if cols[3] != "unread" {
		t.Fatalf("expected unread, got %s", cols[3])
	}
}

func TestReleaseRowRead(t *testing.T) {
	row := ReleaseRow{
		Release: &store.Release{
			IsRead: true,
		},
	}
	cols := row.GetColumns()
	if cols[3] != "read" {
		t.Fatalf("expected read, got %s", cols[3])
	}
}

func TestReleaseSummary(t *testing.T) {
	r := &store.Release{
		TagName:      "v2.0.0",
		Name:         "Major Release",
		Body:         "Breaking changes",
		PublishedAt:  "2025-07-01",
		IsPrerelease: true,
	}

	summary := ReleaseSummary(r)
	if !strings.Contains(summary, "v2.0.0") {
		t.Fatal("missing tag")
	}
	if !strings.Contains(summary, "pre-release") {
		t.Fatal("missing pre-release label")
	}
	if !strings.Contains(summary, "Major Release") {
		t.Fatal("missing name")
	}
	if !strings.Contains(summary, "Breaking changes") {
		t.Fatal("missing body")
	}
}

func TestGroupHeaderRow(t *testing.T) {
	header := GroupHeaderRow{Title: "owner/repo"}
	if header.GetTitle() != "owner/repo" {
		t.Fatalf("unexpected title: %s", header.GetTitle())
	}
	if header.GetId() != "group:owner/repo" {
		t.Fatalf("unexpected id: %s", header.GetId())
	}
}

func TestReleaseSummaryLongBody(t *testing.T) {
	longBody := strings.Repeat("x", 600)
	r := &store.Release{
		TagName: "v1.0.0",
		Body:    longBody,
	}

	summary := ReleaseSummary(r)
	if len(summary) < 500 {
		t.Fatalf("expected long summary, got %d chars", len(summary))
	}
	if !strings.Contains(summary, "...") {
		t.Fatal("expected truncation")
	}
}
