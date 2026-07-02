package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func (s *sqliteStore) InsertVector(ctx context.Context, repoID int64, embedding []float64) error {
	vecJSON, err := json.Marshal(embedding)
	if err != nil {
		return fmt.Errorf("marshal vector: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM repo_vectors WHERE rowid = ?`, repoID)
	if err != nil {
		return fmt.Errorf("delete old vector: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO repo_vectors (rowid, embedding) VALUES (?, ?)`, repoID, string(vecJSON))
	if err != nil {
		return fmt.Errorf("insert vector: %w", err)
	}
	return nil
}

func (s *sqliteStore) SearchVectors(ctx context.Context, queryVec []float64, topK int, threshold float64) ([]VectorMatch, error) {
	vecJSON, err := json.Marshal(queryVec)
	if err != nil {
		return nil, fmt.Errorf("marshal query vector: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT rowid, distance
		FROM repo_vectors
		WHERE embedding MATCH ?
		ORDER BY distance
		LIMIT %d`, topK),
		string(vecJSON))
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()
	var results []VectorMatch
	for rows.Next() {
		var match VectorMatch
		if err := rows.Scan(&match.RepoID, &match.Distance); err != nil {
			return nil, fmt.Errorf("scan vector result: %w", err)
		}
		match.Similarity = 1.0 / (1.0 + match.Distance)
		if match.Similarity < threshold {
			continue
		}
		results = append(results, match)
	}
	return results, rows.Err()
}

func (s *sqliteStore) DeleteVector(ctx context.Context, repoID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repo_vectors WHERE rowid = ?`, repoID)
	return err
}

func (s *sqliteStore) SetVectorIndexedAt(ctx context.Context, repoID int64, t time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE repositories SET vector_indexed_at = ? WHERE id = ?`,
		t.Format(time.RFC3339), repoID)
	return err
}

func (s *sqliteStore) GetRepositoryByID(ctx context.Context, id int64) (*Repository, error) {
	row := s.db.QueryRowContext(ctx, repositoryColumns+` WHERE id = ?`, id)
	r, err := scanRepository(row)
	if err != nil {
		return nil, fmt.Errorf("get repo %d: %w", id, err)
	}
	return r, nil
}

func (s *sqliteStore) ListVectorUnindexed(ctx context.Context, limit int) ([]*Repository, error) {
	query := repositoryColumns + ` WHERE analyzed_at IS NOT NULL AND analysis_failed = 0 AND vector_indexed_at IS NULL ORDER BY full_name`
	if limit > 0 {
		query += ` LIMIT ?`
	}
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.db.QueryContext(ctx, query, limit)
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
