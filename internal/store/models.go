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
	AISummary          string
	AITags             []string
	AIPlatforms        []string
	AICategory         string
	AnalyzedAt         *time.Time
	AnalysisFailed     bool
	CustomDescription  string
	CustomTags         []string
	CustomCategory     string
	CategoryLocked     bool
	SubscribedReleases bool
	LastReleaseFetch   *time.Time
}

type Release struct {
	ID           int64
	RepoID       int64
	RepoFullName string
	TagName      string
	Name         string
	Body         string
	HTMLURL      string
	PublishedAt  string
	IsPrerelease bool
	IsDraft      bool
	IsRead       bool
	Assets       []ReleaseAsset
}

type ReleaseAsset struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
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
	Summary   string
	Tags      []string
	Platforms []string
	Category  string
}

type CustomFields struct {
	Description    string
	Tags           []string
	Category       string
	CategoryLocked bool
}
