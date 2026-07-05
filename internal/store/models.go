package store

import "time"

type Repository struct {
	ID                 int64
	FullName           string
	Name               string
	Description        string
	URL                string
	Language           string
	Homepage           string
	StargazersCount    int
	ForksCount         int
	Topics             []string
	OwnerLogin         string
	OwnerAvatar        string
	StarredAt          string
	RepoUpdatedAt      string    // 仓库在 GitHub 上的最后更新时间 (RFC3339)
	AISummary          string
	AITags             []string
	AIPlatforms        []string
	AICategory         string
	AISearchText       string
	AnalyzedAt         *time.Time
	AnalysisFailed     bool
	CustomDescription  string
	CustomTags         []string
	CustomCategory     string
	CategoryLocked     bool
	VectorIndexedAt    *time.Time
}

// Deprecated: FTSResult was used for FTS5 full-text search results.
// In-memory search now uses ai.SearchHit directly.
// This type is retained for backward compatibility and may be removed.
type FTSResult struct {
	Repo       *Repository
	BM25Score  float64
}

type SearchFilters struct {
	Language       string
	Category       string
	MinStars       int
	MaxStars       int
	Platform       string
	Tags           []string
	Limit          int
	Analyzed       *bool
	AnalysisFailed *bool
}

type Category struct {
	ID        string
	Name      string
	Keywords  []string
	SortOrder int
	IsCustom  bool
	IsHidden  bool
}

type AIResult struct {
	Summary    string
	Tags       []string
	Platforms  []string
	Category   string
	SearchText string
}

type CustomFields struct {
	Description    string
	Tags           []string
	Category       string
	CategoryLocked bool
}

type VectorMatch struct {
	RepoID     int64
	Distance   float64
	Similarity float64
}
