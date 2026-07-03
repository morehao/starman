package components

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/tui/styles"
)

func TestCommandWorkspaceValidateRequiredField(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{
		ID: "sync",
		Params: []CommandParamSpec{{Key: "target", Label: "Target", Type: ParamText, Required: true}},
	})
	if err := w.Validate(); err == nil {
		t.Fatal("expected validation error for required field")
	}
}

func TestCommandWorkspaceValidateNoErrorWhenOptionalOnly(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{
		ID: "sync",
		Params: []CommandParamSpec{{Key: "target", Label: "Target", Type: ParamText, Required: false}},
	})
	if err := w.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestCommandWorkspaceValidateNoCommandSelected(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	if err := w.Validate(); err == nil {
		t.Fatal("expected validation error when no command selected")
	}
}

func TestCommandWorkspaceViewIncludesSections(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{ID: "sync", Label: "Sync", Description: "同步"})
	v := w.View()
	for _, token := range []string{"Sync", "Output"} {
		if !strings.Contains(v, token) {
			t.Fatalf("view missing section %s", token)
		}
	}
}

func TestCommandWorkspaceViewShowsCommandInfo(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{
		ID:          "sync",
		Label:       "Sync",
		Description: "同步仓库",
		Shortcut:    "s",
	})
	v := w.View()
	if !strings.Contains(v, "Sync") {
		t.Fatal("view missing command label")
	}
	if !strings.Contains(v, "同步仓库") {
		t.Fatal("view missing command description")
	}
	if !strings.Contains(v, "s") {
		t.Fatal("view missing command shortcut")
	}
}

func TestCommandWorkspaceParamsReturnsDefaults(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{
		ID: "sync",
		Params: []CommandParamSpec{
			{Key: "target", Label: "Target", Type: ParamText, DefaultValue: "github.com"},
		},
	})
	params := w.Params()
	if params["target"] != "github.com" {
		t.Fatalf("expected default value 'github.com', got '%s'", params["target"])
	}
}

func TestCommandWorkspaceOutputAppend(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{ID: "sync", Label: "Sync"})
	w.AppendOutput("log line 1")
	w.AppendOutput("log line 2")
	v := w.View()
	if !strings.Contains(v, "log line 1") {
		t.Fatal("view missing output line 1")
	}
	if !strings.Contains(v, "log line 2") {
		t.Fatal("view missing output line 2")
	}
}

func TestCommandWorkspaceLayoutMatchesSpec(t *testing.T) {
	w := NewCommandWorkspace(styles.DefaultTheme())
	w.SelectCommand(CommandNode{
		ID:          "sync",
		Label:       "Sync Repositories",
		Description: "拉取 GitHub 星标并合并到本地库",
		Shortcut:    "s",
		Params: []CommandParamSpec{
			{Key: "include_archived", Label: "include archived", Type: ParamBool, Required: false},
			{Key: "concurrency", Label: "concurrency", Type: ParamNumber, Required: false, DefaultValue: "3"},
		},
	})
	w.AppendOutput("fetching starred repos... 340 found")
	v := w.View()

	checks := []string{
		"▍Sync Repositories",
		"[s]",
		"拉取 GitHub 星标",
		"Params",
		"include archived",
		"concurrency",
		"[3]",
		"Output",
		"fetching starred repos",
	}
	for _, c := range checks {
		if !strings.Contains(v, c) {
			t.Fatalf("missing %q in layout\n=== got ===\n%s", c, v)
		}
	}

	if strings.Count(v, "──") < 2 {
		t.Fatalf("missing section separators\n=== got ===\n%s", v)
	}
}
