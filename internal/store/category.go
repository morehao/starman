package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *sqliteStore) ListCategories(ctx context.Context, visibleOnly bool) ([]*Category, error) {
	query := `SELECT id, name, keywords, sort_order, is_custom, is_hidden FROM categories ORDER BY sort_order, id`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var cats []*Category
	for rows.Next() {
		c := &Category{}
		var kwJSON sql.NullString
		var isCustom, isHidden int
		if err := rows.Scan(&c.ID, &c.Name, &kwJSON, &c.SortOrder, &isCustom, &isHidden); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		if kwJSON.Valid {
			if err := json.Unmarshal([]byte(kwJSON.String), &c.Keywords); err != nil {
				return nil, fmt.Errorf("unmarshal category keywords: %w", err)
			}
		}
		c.IsCustom = isCustom != 0
		c.IsHidden = isHidden != 0
		if visibleOnly && c.IsHidden {
			continue
		}
		cats = append(cats, c)
	}
	return cats, rows.Err()
}

func (s *sqliteStore) UpsertCategory(ctx context.Context, c *Category) error {
	kwJSON, _ := json.Marshal(c.Keywords)
	isCustom := 0
	if c.IsCustom {
		isCustom = 1
	}
	isHidden := 0
	if c.IsHidden {
		isHidden = 1
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO categories (id, name, keywords, sort_order, is_custom, is_hidden) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, keywords=excluded.keywords, sort_order=excluded.sort_order, is_custom=excluded.is_custom, is_hidden=excluded.is_hidden`,
		c.ID, c.Name, string(kwJSON), c.SortOrder, isCustom, isHidden)
	if err != nil {
		return fmt.Errorf("upsert category: %w", err)
	}
	return nil
}

func (s *sqliteStore) DeleteCategory(ctx context.Context, id string) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var isCustom int
	err = tx.QueryRowContext(ctx, `SELECT is_custom FROM categories WHERE id = ?`, id).Scan(&isCustom)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("category %s not found", id)
	}
	if err != nil {
		return 0, fmt.Errorf("query category: %w", err)
	}
	if isCustom == 0 {
		return 0, fmt.Errorf("cannot delete built-in category %s", id)
	}

	var affected int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM repositories WHERE custom_category = ?`, id).Scan(&affected)
	if err != nil {
		return 0, fmt.Errorf("count affected repos: %w", err)
	}

	if affected > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE repositories SET custom_category = '', updated_at = datetime('now') WHERE custom_category = ?`, id); err != nil {
			return 0, fmt.Errorf("clear repo categories: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM categories WHERE id = ?`, id); err != nil {
		return 0, fmt.Errorf("delete category: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return affected, nil
}

func Slugify(s string) string {
	return slugify(s)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return strings.Trim(result.String(), "-")
}
