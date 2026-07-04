package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

func (s *sqliteStore) UpsertReposTouchOnly(ctx context.Context, repos []*Repository) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, r := range repos {
		_, err := tx.ExecContext(ctx,
			`UPDATE repositories SET repo_updated_at=?, updated_at=datetime('now') WHERE full_name=?`,
			r.RepoUpdatedAt, r.FullName)
		if err != nil {
			return fmt.Errorf("touch repo %s: %w", r.FullName, err)
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
	var vectorIndexedAt interface{}
	if r.VectorIndexedAt != nil {
		vectorIndexedAt = r.VectorIndexedAt.Format(time.RFC3339)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO repositories (
		id, full_name, name, description, url, language, homepage,
		stargazers_count, forks_count, topics, owner_login, owner_avatar, starred_at, repo_updated_at,
		ai_summary, ai_tags, ai_platforms, ai_category, ai_search_text, analyzed_at, analysis_failed,
		custom_description, custom_tags, custom_category, category_locked,
		subscribed_releases, last_release_fetch, vector_indexed_at
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(id) DO UPDATE SET
		full_name=excluded.full_name, name=excluded.name, description=excluded.description,
		url=excluded.url, language=excluded.language, homepage=excluded.homepage,
		stargazers_count=excluded.stargazers_count, forks_count=excluded.forks_count,
		topics=excluded.topics, owner_login=excluded.owner_login, owner_avatar=excluded.owner_avatar,
		starred_at=excluded.starred_at,
		repo_updated_at=excluded.repo_updated_at,
		ai_summary=excluded.ai_summary, ai_tags=excluded.ai_tags, ai_platforms=excluded.ai_platforms,
		ai_category=excluded.ai_category, ai_search_text=excluded.ai_search_text,
		analyzed_at=excluded.analyzed_at, analysis_failed=excluded.analysis_failed,
		custom_description=excluded.custom_description, custom_tags=excluded.custom_tags,
		custom_category=excluded.custom_category, category_locked=excluded.category_locked,
		subscribed_releases=excluded.subscribed_releases, last_release_fetch=excluded.last_release_fetch,
		vector_indexed_at=excluded.vector_indexed_at,
		updated_at=datetime('now')`,
		r.ID, r.FullName, r.Name, r.Description, r.URL, r.Language, r.Homepage,
		r.StargazersCount, r.ForksCount, string(topicsJSON), r.OwnerLogin, r.OwnerAvatar, r.StarredAt, r.RepoUpdatedAt,
		r.AISummary, string(tagsJSON), string(platJSON), r.AICategory, r.AISearchText, analyzedAt, analysisFailed,
		r.CustomDescription, string(customTagsJSON), r.CustomCategory, categoryLocked,
		subscribed, lastReleaseFetch, vectorIndexedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert repo %s: %w", r.FullName, err)
	}
	return nil
}

func scanRepository(row interface{ Scan(dest ...any) error }) (*Repository, error) {
	r := &Repository{}
	var topicsJSON, tagsJSON, platJSON, customTagsJSON sql.NullString
	var analyzedAt, lastReleaseFetch, vectorIndexedAt sql.NullString
	var customDesc, customCat sql.NullString
	var analysisFailed, categoryLocked, subscribed int
	var searchText, repoUpdatedAt sql.NullString
	err := row.Scan(
		&r.ID, &r.FullName, &r.Name, &r.Description, &r.URL, &r.Language, &r.Homepage,
		&r.StargazersCount, &r.ForksCount, &topicsJSON, &r.OwnerLogin, &r.OwnerAvatar, &r.StarredAt,
		&repoUpdatedAt,
		&r.AISummary, &tagsJSON, &platJSON, &r.AICategory, &searchText, &analyzedAt, &analysisFailed,
		&customDesc, &customTagsJSON, &customCat, &categoryLocked,
		&subscribed, &lastReleaseFetch, &vectorIndexedAt,
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
	if customDesc.Valid {
		r.CustomDescription = customDesc.String
	}
	if customCat.Valid {
		r.CustomCategory = customCat.String
	}
	if searchText.Valid {
		r.AISearchText = searchText.String
	}
	if repoUpdatedAt.Valid {
		r.RepoUpdatedAt = repoUpdatedAt.String
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
	if vectorIndexedAt.Valid {
		t, err := time.Parse(time.RFC3339, vectorIndexedAt.String)
		if err == nil {
			r.VectorIndexedAt = &t
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
	stargazers_count, forks_count, topics, owner_login, owner_avatar, starred_at, repo_updated_at,
	ai_summary, ai_tags, ai_platforms, ai_category, ai_search_text, analyzed_at, analysis_failed,
	custom_description, custom_tags, custom_category, category_locked,
	subscribed_releases, last_release_fetch, vector_indexed_at FROM repositories`

func (s *sqliteStore) ListRepositories(ctx context.Context) ([]*Repository, error) {
	rows, err := s.db.QueryContext(ctx, repositoryColumns+` ORDER BY repo_updated_at DESC NULLS LAST, full_name`)
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
	query := repositoryColumns + ` WHERE analyzed_at IS NULL ORDER BY repo_updated_at DESC NULLS LAST, full_name`
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
	rows, err := s.db.QueryContext(ctx, repositoryColumns+` WHERE COALESCE(NULLIF(custom_category,''), NULLIF(ai_category,''), '其他') = ? ORDER BY repo_updated_at DESC NULLS LAST, full_name`, category)
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
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET ai_summary=?, ai_tags=?, ai_platforms=?, ai_category=?, ai_search_text=?, analyzed_at=strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), analysis_failed=0, updated_at=datetime('now') WHERE id=?`,
		res.Summary, string(tagsJSON), string(platJSON), res.Category, res.SearchText, repoID)
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

	existing, err := s.listReposByFullName(ctx, tx)
	if err != nil {
		return fmt.Errorf("list existing repos: %w", err)
	}

	merged := MergeReposOnSync(rs, existing)

	for _, r := range merged {
		if err := upsertRepoTx(ctx, tx, r); err != nil {
			return err
		}
	}

	if fullSync {
		incomingNames := make(map[string]bool, len(rs))
		for _, r := range rs {
			incomingNames[r.FullName] = true
		}
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

	if _, err := tx.ExecContext(ctx, `DELETE FROM sync_state WHERE key LIKE 'search_cache:%'`); err != nil {
		return fmt.Errorf("clear search cache: %w", err)
	}

	return tx.Commit()
}

func (s *sqliteStore) listReposByFullName(ctx context.Context, tx *sql.Tx) (map[string]*Repository, error) {
	rows, err := tx.QueryContext(ctx, repositoryColumns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*Repository)
	for rows.Next() {
		r, err := scanRepository(rows)
		if err != nil {
			return nil, err
		}
		result[r.FullName] = r
	}
	return result, rows.Err()
}

// Deprecated: SearchFTS used FTS5 for full-text search. Replaced by in-memory search.
// This implementation is retained for backward compatibility and may be removed.
func (s *sqliteStore) SearchFTS(ctx context.Context, query string, filters *SearchFilters) ([]*FTSResult, error) {
	where := "repositories_fts MATCH ?"
	args := []interface{}{query}
	if filters != nil {
		if filters.Language != "" {
			where += " AND r.language = ?"
			args = append(args, filters.Language)
		}
		if filters.Category != "" {
			where += " AND COALESCE(NULLIF(r.custom_category,''), NULLIF(r.ai_category,''), '其他') = ?"
			args = append(args, filters.Category)
		}
		if filters.MinStars > 0 {
			where += " AND r.stargazers_count >= ?"
			args = append(args, filters.MinStars)
		}
		if filters.MaxStars > 0 {
			where += " AND r.stargazers_count <= ?"
			args = append(args, filters.MaxStars)
		}
		if filters.Platform != "" {
			where += " AND r.ai_platforms LIKE ?"
			args = append(args, "%"+filters.Platform+"%")
		}
		if len(filters.Tags) > 0 {
			parts := make([]string, 0, len(filters.Tags))
			for _, tag := range filters.Tags {
				parts = append(parts, `(r.ai_tags LIKE ? OR r.topics LIKE ? OR r.custom_tags LIKE ?)`)
				args = append(args, "%"+tag+"%", "%"+tag+"%", "%"+tag+"%")
			}
			where += " AND (" + strings.Join(parts, " OR ") + ")"
		}
		if filters.Analyzed != nil {
			if *filters.Analyzed {
				where += " AND r.analyzed_at IS NOT NULL AND r.analysis_failed = 0"
			} else {
				where += " AND r.analyzed_at IS NULL"
			}
		}
		if filters.AnalysisFailed != nil && *filters.AnalysisFailed {
			where += " AND r.analyzed_at IS NOT NULL AND r.analysis_failed = 1"
		}
	}
	limit := 50
	if filters != nil && filters.Limit > 0 {
		limit = filters.Limit
	}
	sqlStr := fmt.Sprintf(`SELECT r.id, r.full_name, r.name, r.description, r.url, r.language, r.homepage,
		r.stargazers_count, r.forks_count, r.topics, r.owner_login, r.owner_avatar, r.starred_at,
		r.ai_summary, r.ai_tags, r.ai_platforms, r.ai_category, r.ai_search_text,
		r.analyzed_at, r.analysis_failed,
		r.custom_description, r.custom_tags, r.custom_category, r.category_locked,
		r.subscribed_releases, r.last_release_fetch, r.vector_indexed_at,
		rank as bm25_score
		FROM repositories_fts
		JOIN repositories r ON repositories_fts.rowid = r.id
		WHERE %s
		ORDER BY rank
		LIMIT ?`, where)
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("search fts: %w", err)
	}
	defer rows.Close()
	var results []*FTSResult
	for rows.Next() {
		r := &Repository{}
		var topicsJSON, tagsJSON, platJSON, customTagsJSON sql.NullString
		var analyzedAt, lastReleaseFetch, vectorIndexedAt sql.NullString
		var customDesc, customCat sql.NullString
		var searchText sql.NullString
		var analysisFailed, categoryLocked, subscribed int
		var bm25 float64
		err := rows.Scan(
			&r.ID, &r.FullName, &r.Name, &r.Description, &r.URL, &r.Language, &r.Homepage,
			&r.StargazersCount, &r.ForksCount, &topicsJSON, &r.OwnerLogin, &r.OwnerAvatar, &r.StarredAt,
			&r.AISummary, &tagsJSON, &platJSON, &r.AICategory, &searchText, &analyzedAt, &analysisFailed,
			&customDesc, &customTagsJSON, &customCat, &categoryLocked,
			&subscribed, &lastReleaseFetch, &vectorIndexedAt,
			&bm25,
		)
		if err != nil {
			return nil, fmt.Errorf("scan fts result: %w", err)
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
		if customDesc.Valid {
			r.CustomDescription = customDesc.String
		}
		if customCat.Valid {
			r.CustomCategory = customCat.String
		}
		if searchText.Valid {
			r.AISearchText = searchText.String
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
		if vectorIndexedAt.Valid {
			t, err := time.Parse(time.RFC3339, vectorIndexedAt.String)
			if err == nil {
				r.VectorIndexedAt = &t
			}
		}
		results = append(results, &FTSResult{Repo: r, BM25Score: bm25})
	}
	return results, rows.Err()
}

// Deprecated: RebuildFTSIndex was used to refresh the FTS5 full-text index. Replaced by in-memory search.
// This implementation is retained for backward compatibility and may be removed.
func (s *sqliteStore) RebuildFTSIndex(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO repositories_fts(repositories_fts) VALUES('rebuild')`)
	return err
}
