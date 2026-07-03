package components

import "github.com/morehao/starman/internal/tui/types"

type CommandParamType string

const (
	ParamBool        CommandParamType = "bool"
	ParamEnum        CommandParamType = "enum"
	ParamText        CommandParamType = "text"
	ParamNumber      CommandParamType = "number"
	ParamPath        CommandParamType = "path"
	ParamMultiSelect CommandParamType = "multi-select"
)

type CommandParamSpec struct {
	Key          string
	Label        string
	Type         CommandParamType
	Required     bool
	DefaultValue string
	Options      []string
}

type CommandNode struct {
	ID          string
	Label       string
	Description string
	Shortcut    string
	Page        types.PageID
	Group       string
	ParentID    string
	Params      []CommandParamSpec
}

func DefaultCommandCatalog() []CommandNode {
	return []CommandNode{
		{ID: "search", Label: "Search", Description: "搜索仓库", Shortcut: "/", Page: types.PageSearch, Group: "发现"},
		{ID: "trending", Label: "Trending", Description: "趋势仓库", Shortcut: "t", Page: types.PageTrending, Group: "发现"},
		{ID: "repo-list", Label: "Repo List", Description: "仓库列表", Shortcut: "r", Page: types.PageRepoList, Group: "整理"},
		{ID: "tag", Label: "Tag", Description: "标签管理", Shortcut: "g", Page: types.PageTag, Group: "整理"},
		{ID: "categorize", Label: "Categorize", Description: "分类管理", Shortcut: "c", Page: types.PageCategorize, Group: "整理"},
		{ID: "stats", Label: "Stats", Description: "统计信息", Shortcut: "S", Page: types.PageStats, Group: "整理"},
		{ID: "sync", Label: "Sync", Description: "同步仓库", Shortcut: "s", Page: types.PageSync, Group: "处理"},
		{ID: "analyze", Label: "Analyze", Description: "AI 分析", Shortcut: "a", Page: types.PageAnalyze, Group: "处理"},
		{ID: "generate", Label: "Generate", Description: "生成清单", Shortcut: "G", Page: types.PageGenerate, Group: "处理"},
		{ID: "release", Label: "Release", Description: "发布管理", Shortcut: "R", Page: types.PageRelease, Group: "处理"},
		{ID: "backup", Label: "Backup", Description: "备份导入导出", Shortcut: "b", Page: types.PageBackup, Group: "处理"},
		{ID: "dashboard", Label: "Dashboard", Description: "总览", Shortcut: "1", Page: types.PageDashboard, Group: "系统"},
		{ID: "config", Label: "Config", Description: "配置管理", Shortcut: "C", Page: types.PageConfig, Group: "系统"},
	}
}

func DefaultPinnedCommandIDs() []string {
	return []string{"dashboard", "config", "sync", "search"}
}
