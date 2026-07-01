package store

import (
	"context"
	"database/sql"
	"errors"
)

var ErrSyncStateNotFound = errors.New("sync state key not found")

func (s *sqliteStore) GetSyncState(ctx context.Context, key string) (string, error) {
	var val string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM sync_state WHERE key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", ErrSyncStateNotFound
	}
	return val, err
}

func (s *sqliteStore) SetSyncState(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
