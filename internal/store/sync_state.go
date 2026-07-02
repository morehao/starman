package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
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

type SyncStats struct {
	TotalSyncCount int       `json:"total_sync_count"`
	LastSync       time.Time `json:"last_sync"`
	LastDuration   string    `json:"last_duration"`
	LastRepoCount  int       `json:"last_repo_count"`
	LastNewCount   int       `json:"last_new_count"`
	LastErrorCount int       `json:"last_error_count"`
}

func (s *sqliteStore) SaveSyncStats(ctx context.Context, stats *SyncStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal sync stats: %w", err)
	}
	return s.SetSyncState(ctx, "sync_stats", string(data))
}

func (s *sqliteStore) GetSyncStats(ctx context.Context) (*SyncStats, error) {
	val, err := s.GetSyncState(ctx, "sync_stats")
	if err == ErrSyncStateNotFound {
		return &SyncStats{}, nil
	}
	if err != nil {
		return nil, err
	}
	var stats SyncStats
	if err := json.Unmarshal([]byte(val), &stats); err != nil {
		return &SyncStats{}, nil
	}
	return &stats, nil
}

func (s *sqliteStore) IncrementSyncCount(ctx context.Context) error {
	stats, err := s.GetSyncStats(ctx)
	if err != nil {
		return err
	}
	stats.TotalSyncCount++
	return s.SaveSyncStats(ctx, stats)
}
