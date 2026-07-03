package tui

import (
	"github.com/morehao/starman/internal/tui/types"
)

type PageID = types.PageID

const (
	PageDashboard  = types.PageDashboard
	PageSearch     = types.PageSearch
	PageRepoList   = types.PageRepoList
	PageTrending   = types.PageTrending
	PageSync       = types.PageSync
	PageAnalyze    = types.PageAnalyze
	PageTag        = types.PageTag
	PageCategorize = types.PageCategorize
	PageStats      = types.PageStats
	PageRelease    = types.PageRelease
	PageGenerate   = types.PageGenerate
	PageBackup     = types.PageBackup
	PageConfig     = types.PageConfig
	PageRepoDetail = types.PageRepoDetail
)

type NavigatedMsg = types.NavigatedMsg

type StatusLevel = types.StatusLevel

const (
	LevelInfo    = types.LevelInfo
	LevelSuccess = types.LevelSuccess
	LevelWarning = types.LevelWarning
	LevelError   = types.LevelError
)

type StatusMsg = types.StatusMsg

type TaskStartedMsg = types.TaskStartedMsg
type TaskProgressMsg = types.TaskProgressMsg
type TaskDoneMsg = types.TaskDoneMsg

type RepoSelectedMsg = types.RepoSelectedMsg

type TickMsg = types.TickMsg

type CommandSelectedMsg = types.CommandSelectedMsg

type UIState = types.UIState

const (
	StateNormal  = types.StateNormal
	StatePalette = types.StatePalette
)

type FocusPane = types.FocusPane

const (
	FocusSidebar   = types.FocusSidebar
	FocusWorkspace = types.FocusWorkspace
)
