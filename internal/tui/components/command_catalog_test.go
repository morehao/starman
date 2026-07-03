package components

import (
	"testing"

	"github.com/morehao/starman/internal/tui/types"
)

func TestDefaultCommandCatalogHasCoreCommands(t *testing.T) {
	catalog := DefaultCommandCatalog()
	if len(catalog) == 0 {
		t.Fatal("catalog should not be empty")
	}
	want := map[string]bool{"sync": false, "analyze": false, "generate": false, "backup": false}
	for _, node := range catalog {
		if _, ok := want[node.ID]; ok {
			want[node.ID] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("missing command %s", id)
		}
	}
}

func TestDefaultPinnedCommandIDsIncludesDashboardAndConfig(t *testing.T) {
	pinned := DefaultPinnedCommandIDs()
	if len(pinned) == 0 {
		t.Fatal("pinned should not be empty")
	}
	want := map[string]bool{"dashboard": false, "config": false}
	for _, id := range pinned {
		if _, ok := want[id]; ok {
			want[id] = true
		}
	}
	for id, seen := range want {
		if !seen {
			t.Fatalf("missing pinned command %s", id)
		}
	}
}

func TestDefaultCommandCatalogMatchesSidebarGroups(t *testing.T) {
	catalog := DefaultCommandCatalog()
	expected := []struct {
		id       string
		label    string
		shortcut string
		page     types.PageID
		group    string
	}{
		{"search", "Search", "/", types.PageSearch, "发现"},
		{"trending", "Trending", "t", types.PageTrending, "发现"},
		{"repo-list", "Repo List", "r", types.PageRepoList, "整理"},
		{"tag", "Tag", "g", types.PageTag, "整理"},
		{"categorize", "Categorize", "c", types.PageCategorize, "整理"},
		{"stats", "Stats", "S", types.PageStats, "整理"},
		{"sync", "Sync", "s", types.PageSync, "处理"},
		{"analyze", "Analyze", "a", types.PageAnalyze, "处理"},
		{"generate", "Generate", "G", types.PageGenerate, "处理"},
		{"release", "Release", "R", types.PageRelease, "处理"},
		{"backup", "Backup", "b", types.PageBackup, "处理"},
		{"dashboard", "Dashboard", "1", types.PageDashboard, "系统"},
		{"config", "Config", "C", types.PageConfig, "系统"},
	}

	for _, exp := range expected {
		found := false
		for _, node := range catalog {
			if node.ID == exp.id {
				found = true
				if node.Label != exp.label {
					t.Errorf("command %s: label = %q, want %q", exp.id, node.Label, exp.label)
				}
				if node.Shortcut != exp.shortcut {
					t.Errorf("command %s: shortcut = %q, want %q", exp.id, node.Shortcut, exp.shortcut)
				}
				if node.Page != exp.page {
					t.Errorf("command %s: page = %v, want %v", exp.id, node.Page, exp.page)
				}
				if node.Group != exp.group {
					t.Errorf("command %s: group = %q, want %q", exp.id, node.Group, exp.group)
				}
				break
			}
		}
		if !found {
			t.Fatalf("missing command %s in catalog", exp.id)
		}
	}
}

func TestCommandParamTypes(t *testing.T) {
	types := []CommandParamType{ParamBool, ParamEnum, ParamText, ParamNumber, ParamPath, ParamMultiSelect}
	for _, pt := range types {
		if pt == "" {
			t.Error("command param type should not be empty string")
		}
	}
}

func TestCommandNodeStruct(t *testing.T) {
	node := CommandNode{
		ID:          "test",
		Label:       "Test",
		Description: "A test command",
		Shortcut:    "T",
		Page:        types.PageDashboard,
		Group:       "测试",
		ParentID:    "parent",
		Params: []CommandParamSpec{
			{Key: "verbose", Label: "Verbose", Type: ParamBool, Required: false, DefaultValue: "false"},
		},
	}
	if node.ID != "test" {
		t.Errorf("ID = %q, want %q", node.ID, "test")
	}
	if node.Label != "Test" {
		t.Errorf("Label = %q, want %q", node.Label, "Test")
	}
	if node.Description != "A test command" {
		t.Errorf("Description = %q, want %q", node.Description, "A test command")
	}
	if node.Params[0].Key != "verbose" {
		t.Errorf("Params[0].Key = %q, want %q", node.Params[0].Key, "verbose")
	}
	if node.Params[0].Options != nil {
		t.Error("Params[0].Options should default to nil")
	}
}
