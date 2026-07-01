package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *sqliteStore) UpsertRepository(ctx context.Context, r *Repository) error {
	return s.UpsertRepositories(ctx, []*Repository{r})
}

func (s *sqliteStore) UpsertRepositories(ctx context.Context, rs []*Repository) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	for _, r := range rs {
		if err := upsertRepoTx(ctx, tx, r); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func upsertRepoTx(ctx context.Context, tx *sql.Tx, r *Repository) error {
	topicsJSON, _ := json.Marshal(r.Topics)
	tagsJSON, _ := json.Marshal(r.AITags)
	platJSON, _ := json.Marshal(r.AIPlatforms)
	customTagsJSON, _ := json.Marshal(r.CustomTags)
	var analyzedAt interface{}
	if r.AnalyzedAt != nil {
		analyzedAt = r.AnalyzedAt.Format(time.RFC3339)
	}
	analysisFailed := 0
	if r.AnalysisFailed {
		analysisFailed = 1
	}
	categoryLocked := 0
	if r.CategoryLocked {
		categoryLocked = 1
	}
	subscribed := 0
	if r.SubscribedReleases {
		subscribed = 1
	}
	var lastReleaseFetch interface{}
	if r.LastReleaseFetch != nil {
		lastReleaseFetch = r.LastReleaseFetch.Format(time.RFC3339)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO repositories (
		id, full_name, name, description, url, language, homepage,
		stargazers_count, forks_count, topics, owner_login, owner_avatar, starred_at,
		ai_summary, ai_tags, ai_platforms, ai_category, analyzed_at, analysis_failed,
		custom_description, custom_tags, custom_category, category_locked,
		subscribed_releases, last_release_fetch
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(id) DO UPDATE SET
		full_name=excluded.full_name, name=excluded.name, description=excluded.description,
		url=excluded.url, language=excluded.language, homepage=excluded.homepage,
		stargazers_count=excluded.stargazers_count, forks_count=excluded.forks_count,
		topics=excluded.topics, owner_login=excluded.owner_login, owner_avatar=excluded.owner_avatar,
		starred_at=excluded.starred_at,
		ai_summary=excluded.ai_summary, ai_tags=excluded.ai_tags, ai_platforms=excluded.ai_platforms,
		ai_category=excluded.ai_category, analyzed_at=excluded.analyzed_at, analysis_failed=excluded.analysis_failed,
		custom_description=excluded.custom_description, custom_tags=excluded.custom_tags,
		custom_category=excluded.custom_category, category_locked=excluded.category_locked,
		subscribed_releases=excluded.subscribed_releases, last_release_fetch=excluded.last_release_fetch,
		updated_at=datetime('now')`,
		r.ID, r.FullName, r.Name, r.Description, r.URL, r.Language, r.Homepage,
		r.StargazersCount, r.ForksCount, string(topicsJSON), r.OwnerLogin, r.OwnerAvatar, r.StarredAt,
		r.AISummary, string(tagsJSON), string(platJSON), r.AICategory, analyzedAt, analysisFailed,
		r.CustomDescription, string(customTagsJSON), r.CustomCategory, categoryLocked,
		subscribed, lastReleaseFetch,
	)
	if err != nil {
		return fmt.Errorf("upsert repo %s: %w", r.FullName, err)
	}
	return nil
}

func scanRepository(row interface{ Scan(dest ...any) error }) (*Repository, error) {
	r := &Repository{}
	var topicsJSON, tagsJSON, platJSON, customTagsJSON sql.NullString
	var analyzedAt, lastReleaseFetch sql.NullString
	var analysisFailed, categoryLocked, subscribed int
	err := row.Scan(
		&r.ID, &r.FullName, &r.Name, &r.Description, &r.URL, &r.Language, &r.Homepage,
		&r.StargazersCount, &r.ForksCount, &topicsJSON, &r.OwnerLogin, &r.OwnerAvatar, &r.StarredAt,
		&r.AISummary, &tagsJSON, &platJSON, &r.AICategory, &analyzedAt, &analysisFailed,
		&r.CustomDescription, &customTagsJSON, &r.CustomCategory, &categoryLocked,
		&subscribed, &lastReleaseFetch,
	)
	if err != nil {
		return nil, err
	}
	if topicsJSON.Valid {
		json.Unmarshal([]byte(topicsJSON.String), &r.Topics)
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &r.AITags)
	}
	if platJSON.Valid {
		json.Unmarshal([]byte(platJSON.String), &r.AIPlatforms)
	}
	if customTagsJSON.Valid {
		json.Unmarshal([]byte(customTagsJSON.String), &r.CustomTags)
	}
	if analyzedAt.Valid {
		t, err := time.Parse(time.RFC3339, analyzedAt.String)
		if err == nil {
			r.AnalyzedAt = &t
		}
	}
	r.AnalysisFailed = analysisFailed != 0
	r.CategoryLocked = categoryLocked != 0
	r.SubscribedReleases = subscribed != 0
	if lastReleaseFetch.Valid {
		t, err := time.Parse(time.RFC3339, lastReleaseFetch.String)
		if err == nil {
			r.LastReleaseFetch = &t
		}
	}
	return r, nil
}

func (s *sqliteStore) GetRepository(ctx context.Context, fullName string) (*Repository, error) {
	row := s.db.QueryRowContext(ctx, repositoryColumns+` WHERE full_name = ?`, fullName)
	r, err := scanRepository(row)
	if err != nil {
		return nil, fmt.Errorf("get repo %s: %w", fullName, err)
	}
	return r, nil
}

const repositoryColumns = `SELECT id, full_name, name, description, url, language, homepage,
	stargazers_count, forks_count, topics, owner_login, owner_avatar, starred_at,
	ai_summary, ai_tags, ai_platforms, ai_category, analyzed_at, analysis_failed,
	custom_description, custom_tags, custom_category, category_locked,
	subscribed_releases, last_release_fetch FROM repositories`

func (s *sqliteStore) ListRepositories(ctx context.Context) ([]*Repository, error) {
	rows, err := s.db.QueryContext(ctx, repositoryColumns+` ORDER BY full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []*Repository
	for rows.Next() {
		r, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *sqliteStore) ListUnanalyzed(ctx context.Context, limit int) ([]*Repository, error) {
	query := repositoryColumns + ` WHERE analyzed_at IS NULL ORDER BY full_name`
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.db.QueryContext(ctx, query+` LIMIT ?`, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []*Repository
	for rows.Next() {
		r, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *sqliteStore) ListByCategory(ctx context.Context, category string) ([]*Repository, error) {
	rows, err := s.db.QueryContext(ctx, repositoryColumns+` WHERE COALESCE(NULLIF(custom_category,''), NULLIF(ai_category,''), '其他') = ? ORDER BY full_name`, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var repos []*Repository
	for rows.Next() {
		r, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		repos = append(repos, r)
	}
	return repos, rows.Err()
}

func (s *sqliteStore) UpdateAIResult(ctx context.Context, repoID int64, res *AIResult) error {
	tagsJSON, _ := json.Marshal(res.Tags)
	platJSON, _ := json.Marshal(res.Platforms)
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET ai_summary=?, ai_tags=?, ai_platforms=?, ai_category=?, analyzed_at=strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), analysis_failed=0, updated_at=datetime('now') WHERE id=?`,
		res.Summary, string(tagsJSON), string(platJSON), res.Category, repoID)
	return err
}

func (s *sqliteStore) UpdateCustomFields(ctx context.Context, repoID int64, f *CustomFields) error {
	tagsJSON, _ := json.Marshal(f.Tags)
	locked := 0
	if f.CategoryLocked {
		locked = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET custom_description=?, custom_tags=?, custom_category=?, category_locked=?, updated_at=datetime('now') WHERE id=?`,
		f.Description, string(tagsJSON), f.Category, locked, repoID)
	return err
}

func (s *sqliteStore) SetAnalysisFailed(ctx context.Context, repoID int64, failed bool) error {
	v := 0
	if failed {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET analysis_failed=?, updated_at=datetime('now') WHERE id=?`, v, repoID)
	return err
}

func (s *sqliteStore) DeleteAllRepositories(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repositories`)
	return err
}

func (s *sqliteStore) UpsertReposOnSync(ctx context.Context, rs []*Repository, fullSync bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	incomingNames := make(map[string]bool, len(rs))
	for _, r := range rs {
		incomingNames[r.FullName] = true
		topicsJSON, _ := json.Marshal(r.Topics)
		var existingID int64
		var analyzedAt sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT id, analyzed_at FROM repositories WHERE full_name = ?`, r.FullName).Scan(&existingID, &analyzedAt)
		if err == sql.ErrNoRows {
			tagsJSON, _ := json.Marshal(r.AITags)
			platJSON, _ := json.Marshal(r.AIPlatforms)
			_, err = tx.ExecContext(ctx, `INSERT INTO repositories (id, full_name, name, description, url, language, homepage, stargazers_count, forks_count, topics, owner_login, owner_avatar, starred_at, ai_tags, ai_platforms, ai_summary, ai_category) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				r.ID, r.FullName, r.Name, r.Description, r.URL, r.Language, r.Homepage,
				r.StargazersCount, r.ForksCount, string(topicsJSON), r.OwnerLogin, r.OwnerAvatar, r.StarredAt,
				string(tagsJSON), string(platJSON), r.AISummary, r.AICategory)
			if err != nil {
				return fmt.Errorf("insert repo %s: %w", r.FullName, err)
			}
		} else if err != nil {
			return fmt.Errorf("query existing %s: %w", r.FullName, err)
		} else {
			r.ID = existingID
			_, err = tx.ExecContext(ctx, `UPDATE repositories SET name=?, description=?, url=?, language=?, homepage=?, stargazers_count=?, forks_count=?, topics=?, owner_login=?, owner_avatar=?, starred_at=?, updated_at=datetime('now') WHERE id=?`,
				r.Name, r.Description, r.URL, r.Language, r.Homepage,
				r.StargazersCount, r.ForksCount, string(topicsJSON), r.OwnerLogin, r.OwnerAvatar, r.StarredAt,
				existingID)
			if err != nil {
				return fmt.Errorf("update repo %s: %w", r.FullName, err)
			}
		}
		_ = analyzedAt
	}

	if fullSync {
		rows, err := tx.QueryContext(ctx, `SELECT full_name FROM repositories`)
		if err != nil {
			return fmt.Errorf("query all repos for full sync: %w", err)
		}
		var toDelete []string
		for rows.Next() {
			var fn string
			if err := rows.Scan(&fn); err != nil {
				rows.Close()
				return err
			}
			if !incomingNames[fn] {
				toDelete = append(toDelete, fn)
			}
		}
		rows.Close()
		for _, fn := range toDelete {
			if _, err := tx.ExecContext(ctx, `DELETE FROM repositories WHERE full_name = ?`, fn); err != nil {
				return fmt.Errorf("delete repo %s: %w", fn, err)
			}
		}
	}
	return tx.Commit()
}
