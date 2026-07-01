package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/morehao/starman/internal/store"
)

type Backup struct {
	Version      int                 `json:"version"`
	ExportedAt   string              `json:"exported_at"`
	Repositories []*store.Repository `json:"repositories"`
	Releases     []*store.Release    `json:"releases"`
	Categories   []*store.Category   `json:"categories"`
}

type ImportMode string

const (
	ImportMerge   ImportMode = "merge"
	ImportReplace ImportMode = "replace"
)

func ExportJSON(ctx context.Context, s store.Store) ([]byte, error) {
	repos, err := s.ListRepositories(ctx)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	releases, err := s.ListUnreadReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("list releases: %w", err)
	}
	cats, err := s.ListCategories(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	b := Backup{
		Version:      1,
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
		Repositories: repos,
		Releases:     releases,
		Categories:   cats,
	}
	return json.MarshalIndent(b, "", "  ")
}

func ImportJSON(ctx context.Context, s store.Store, data []byte, mode ImportMode) error {
	var b Backup
	if err := json.Unmarshal(data, &b); err != nil {
		return fmt.Errorf("parse backup: %w", err)
	}
	if mode == ImportReplace {
		if err := s.UpsertRepositories(ctx, nil); err != nil {
			return err
		}
	}
	for _, c := range b.Categories {
		if err := s.UpsertCategory(ctx, c); err != nil {
			return fmt.Errorf("import category %s: %w", c.ID, err)
		}
	}
	for _, r := range b.Repositories {
		if err := s.UpsertRepository(ctx, r); err != nil {
			return fmt.Errorf("import repo %s: %w", r.FullName, err)
		}
	}
	return nil
}
