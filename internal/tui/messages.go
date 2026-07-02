package tui

import (
	"time"

	_ "github.com/charmbracelet/bubbles"
)

type PageID int

const (
	PageDashboard PageID = iota
	PageSearch
	PageRepoList
	PageTrending
	PageSync
	PageAnalyze
	PageTag
	PageCategorize
	PageStats
	PageRelease
	PageGenerate
	PageBackup
	PageConfig
	PageRepoDetail
)

type NavigatedMsg struct{ Page PageID }

type StatusLevel int

const (
	LevelInfo StatusLevel = iota
	LevelSuccess
	LevelWarning
	LevelError
)

type StatusMsg struct {
	Text    string
	Level   StatusLevel
	Timeout time.Duration
}

type TaskStartedMsg  struct{ ID, Label string }
type TaskProgressMsg struct{ ID string; Current, Total int }
type TaskDoneMsg     struct{ ID string; Err error }

type RepoSelectedMsg struct{ FullName string }

type TickMsg time.Time
