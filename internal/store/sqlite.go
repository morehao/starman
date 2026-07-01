package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

func Open(dbPath string) (Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &sqliteStore{db: db}
	if err := s.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}

func (s *sqliteStore) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS repositories (
			id                  INTEGER PRIMARY KEY,
			full_name           TEXT NOT NULL UNIQUE,
			name                TEXT NOT NULL,
			description         TEXT,
			url                 TEXT NOT NULL,
			language            TEXT,
			homepage            TEXT,
			stargazers_count    INTEGER DEFAULT 0,
			forks_count         INTEGER DEFAULT 0,
			topics              TEXT,
			owner_login         TEXT,
			owner_avatar        TEXT,
			starred_at          TEXT,
			ai_summary          TEXT,
			ai_tags             TEXT,
			ai_platforms        TEXT,
			ai_category         TEXT,
			analyzed_at         TEXT,
			analysis_failed     INTEGER DEFAULT 0,
			custom_description  TEXT DEFAULT '',
			custom_tags         TEXT DEFAULT '[]',
			custom_category     TEXT DEFAULT '',
			category_locked     INTEGER DEFAULT 0,
			subscribed_releases INTEGER DEFAULT 0,
			last_release_fetch  TEXT,
			created_at          TEXT DEFAULT (datetime('now')),
			updated_at          TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS releases (
			id              INTEGER PRIMARY KEY,
			repo_id         INTEGER NOT NULL,
			repo_full_name  TEXT NOT NULL,
			tag_name        TEXT NOT NULL,
			name            TEXT,
			body            TEXT,
			html_url        TEXT,
			published_at    TEXT,
			is_prerelease   INTEGER DEFAULT 0,
			is_draft        INTEGER DEFAULT 0,
			is_read         INTEGER DEFAULT 0,
			assets          TEXT,
			fetched_at      TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (repo_id) REFERENCES repositories(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			keywords    TEXT,
			sort_order  INTEGER DEFAULT 0,
			is_custom   INTEGER DEFAULT 0,
			is_hidden   INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS sync_state (
			key   TEXT PRIMARY KEY,
			value TEXT
		)`,
	}
	for _, st := range stmts {
		if _, err := s.db.ExecContext(ctx, st); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return s.seedCategories(ctx)
}

var defaultCategories = []Category{
	{ID: "web-app", Name: "Web 应用", Keywords: []string{"web", "frontend", "html", "css", "react", "vue", "frontend-framework"}, SortOrder: 1},
	{ID: "mobile-app", Name: "移动应用", Keywords: []string{"mobile", "ios", "android", "react-native", "flutter", "swift", "kotlin"}, SortOrder: 2},
	{ID: "desktop-app", Name: "桌面应用", Keywords: []string{"desktop", "electron", "tauri", "qt", "gtk"}, SortOrder: 3},
	{ID: "database", Name: "数据库", Keywords: []string{"database", "sql", "nosql", "redis", "postgresql", "mysql"}, SortOrder: 4},
	{ID: "ai-ml", Name: "AI 机器学习", Keywords: []string{"ai", "machine-learning", "deep-learning", "llm", "nlp", "pytorch", "tensorflow"}, SortOrder: 5},
	{ID: "dev-tools", Name: "开发工具", Keywords: []string{"cli", "build-tool", "linter", "debugger", "ide", "devtools"}, SortOrder: 6},
	{ID: "security", Name: "安全工具", Keywords: []string{"security", "crypto", "vulnerability", "pentest"}, SortOrder: 7},
	{ID: "game", Name: "游戏", Keywords: []string{"game", "engine", "graphics", "shader"}, SortOrder: 8},
	{ID: "design", Name: "设计工具", Keywords: []string{"design", "ui", "color", "font", "icon"}, SortOrder: 9},
	{ID: "productivity", Name: "效率工具", Keywords: []string{"productivity", "automation", "workflow"}, SortOrder: 10},
	{ID: "education", Name: "教育学习", Keywords: []string{"education", "tutorial", "documentation", "learning"}, SortOrder: 11},
	{ID: "social", Name: "社交网络", Keywords: []string{"social", "chat", "forum", "community"}, SortOrder: 12},
	{ID: "data-analysis", Name: "数据分析", Keywords: []string{"analytics", "visualization", "dashboard", "etl"}, SortOrder: 13},
	{ID: "others", Name: "其他", Keywords: nil, SortOrder: 99},
}

func (s *sqliteStore) seedCategories(ctx context.Context) error {
	for _, c := range defaultCategories {
		kwJSON, _ := json.Marshal(c.Keywords)
		isCustom := 0
		if c.IsCustom {
			isCustom = 1
		}
		_, err := s.db.ExecContext(ctx,
			`INSERT OR IGNORE INTO categories (id, name, keywords, sort_order, is_custom) VALUES (?, ?, ?, ?, ?)`,
			c.ID, c.Name, string(kwJSON), c.SortOrder, isCustom)
		if err != nil {
			return fmt.Errorf("seed category %s: %w", c.ID, err)
		}
	}
	return nil
}
