package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

const categoryColumns = `SELECT id, name, keywords, sort_order, is_custom, is_hidden FROM categories`

func scanCategory(row interface{ Scan(dest ...any) error }) (*Category, error) {
	c := &Category{}
	var keywordsJSON sql.NullString
	var isCustom, isHidden int
	err := row.Scan(&c.ID, &c.Name, &keywordsJSON, &c.SortOrder, &isCustom, &isHidden)
	if err != nil {
		return nil, err
	}
	if keywordsJSON.Valid {
		json.Unmarshal([]byte(keywordsJSON.String), &c.Keywords)
	}
	c.IsCustom = isCustom != 0
	c.IsHidden = isHidden != 0
	return c, nil
}

func (s *sqliteStore) ListCategories(ctx context.Context, visibleOnly bool) ([]*Category, error) {
	q := categoryColumns
	if visibleOnly {
		q += ` WHERE is_hidden = 0`
	}
	q += ` ORDER BY sort_order, name`
	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cats []*Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
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
	_, err := s.db.ExecContext(ctx, `INSERT INTO categories (id, name, keywords, sort_order, is_custom, is_hidden)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, keywords=excluded.keywords, sort_order=excluded.sort_order, is_custom=excluded.is_custom, is_hidden=excluded.is_hidden`,
		c.ID, c.Name, string(kwJSON), c.SortOrder, isCustom, isHidden)
	if err != nil {
		return fmt.Errorf("upsert category: %w", err)
	}
	return nil
}

func (s *sqliteStore) DeleteCategory(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM categories WHERE id = ? AND is_custom = 1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("cannot delete default or non-existent category: %s", id)
	}
	return nil
}
