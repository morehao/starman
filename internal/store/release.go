package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

const releaseColumns = `SELECT id, repo_id, repo_full_name, tag_name, name, body, html_url,
	published_at, is_prerelease, is_draft, is_read, assets FROM releases`

func scanRelease(row interface{ Scan(dest ...any) error }) (*Release, error) {
	r := &Release{}
	var assetsJSON sql.NullString
	var isPrerelease, isDraft, isRead int
	err := row.Scan(&r.ID, &r.RepoID, &r.RepoFullName, &r.TagName, &r.Name, &r.Body, &r.HTMLURL,
		&r.PublishedAt, &isPrerelease, &isDraft, &isRead, &assetsJSON)
	if err != nil {
		return nil, err
	}
	r.IsPrerelease = isPrerelease != 0
	r.IsDraft = isDraft != 0
	r.IsRead = isRead != 0
	if assetsJSON.Valid {
		json.Unmarshal([]byte(assetsJSON.String), &r.Assets)
	}
	return r, nil
}

func (s *sqliteStore) UpsertRelease(ctx context.Context, r *Release) error {
	assetsJSON, _ := json.Marshal(r.Assets)
	isPrerelease := 0
	if r.IsPrerelease {
		isPrerelease = 1
	}
	isDraft := 0
	if r.IsDraft {
		isDraft = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO releases (id, repo_id, repo_full_name, tag_name, name, body, html_url, published_at, is_prerelease, is_draft, is_read, assets)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			repo_id=excluded.repo_id, repo_full_name=excluded.repo_full_name, tag_name=excluded.tag_name,
			name=excluded.name, body=excluded.body, html_url=excluded.html_url, published_at=excluded.published_at,
			is_prerelease=excluded.is_prerelease, is_draft=excluded.is_draft, assets=excluded.assets`,
		r.ID, r.RepoID, r.RepoFullName, r.TagName, r.Name, r.Body, r.HTMLURL,
		r.PublishedAt, isPrerelease, isDraft, 0, string(assetsJSON))
	if err != nil {
		return fmt.Errorf("upsert release: %w", err)
	}
	return nil
}

func (s *sqliteStore) ListUnreadReleases(ctx context.Context) ([]*Release, error) {
	rows, err := s.db.QueryContext(ctx, releaseColumns+` WHERE is_read = 0 ORDER BY published_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rels []*Release
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *sqliteStore) ListAllReleases(ctx context.Context) ([]*Release, error) {
	rows, err := s.db.QueryContext(ctx, releaseColumns+` ORDER BY published_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rels []*Release
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *sqliteStore) ListReleasesByRepo(ctx context.Context, repoFullName string) ([]*Release, error) {
	rows, err := s.db.QueryContext(ctx, releaseColumns+` WHERE repo_full_name = ? ORDER BY published_at DESC`, repoFullName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rels []*Release
	for rows.Next() {
		r, err := scanRelease(rows)
		if err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *sqliteStore) MarkReleaseRead(ctx context.Context, releaseID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE releases SET is_read = 1 WHERE id = ?`, releaseID)
	return err
}

func (s *sqliteStore) MarkAllReleasesRead(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `UPDATE releases SET is_read = 1 WHERE is_read = 0`)
	return err
}

func (s *sqliteStore) SetReleaseSubscription(ctx context.Context, repoFullName string, subscribed bool) error {
	v := 0
	if subscribed {
		v = 1
	}
	res, err := s.db.ExecContext(ctx, `UPDATE repositories SET subscribed_releases = ?, updated_at=datetime('now') WHERE full_name = ?`, v, repoFullName)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("repository %s not found", repoFullName)
	}
	return nil
}

func (s *sqliteStore) UpdateReleaseWatermark(ctx context.Context, repoID int64, t time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET last_release_fetch = ?, updated_at=datetime('now') WHERE id = ?`, t.Format(time.RFC3339), repoID)
	return err
}
