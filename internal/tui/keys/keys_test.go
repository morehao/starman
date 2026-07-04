package keys

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/config"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()
	if km.Quit.Keys()[0] != "q" {
		t.Errorf("expected Quit='q', got %v", km.Quit.Keys())
	}
	if km.Sync.Keys()[0] != "s" {
		t.Errorf("expected Sync='s', got %v", km.Sync.Keys())
	}
	if km.Refresh.Keys()[0] != "r" {
		t.Errorf("expected Refresh='r', got %v", km.Refresh.Keys())
	}
}

func TestNewKeyMap_OverrideQuitAndSync(t *testing.T) {
	cfg := &config.TUIKeybindings{
		Universal: config.UniversalKeybindings{Quit: "Q"},
		Stars:     config.StarsKeybindings{Sync: "S"},
	}
	km := NewKeyMap(cfg)
	if km.Quit.Keys()[0] != "Q" {
		t.Errorf("expected Quit='Q', got %v", km.Quit.Keys())
	}
	if km.Sync.Keys()[0] != "S" {
		t.Errorf("expected Sync='S', got %v", km.Sync.Keys())
	}
}

func TestNewKeyMap_KeepsDefaultRefresh(t *testing.T) {
	cfg := &config.TUIKeybindings{
		Universal: config.UniversalKeybindings{Quit: "ctrl+c"},
	}
	km := NewKeyMap(cfg)
	if km.Refresh.Keys()[0] != "r" {
		t.Errorf("expected Refresh='r', got %v", km.Refresh.Keys())
	}
}

func TestNewKeyMap_NilConfig(t *testing.T) {
	km := NewKeyMap(nil)
	if km.Quit.Keys()[0] != "q" {
		t.Errorf("expected default Quit='q', got %v", km.Quit.Keys())
	}
}

func TestNewKeyMap_OverrideSearchAndCommand(t *testing.T) {
	cfg := &config.TUIKeybindings{
		Universal: config.UniversalKeybindings{Search: "ctrl+f", Command: "ctrl+p"},
	}
	km := NewKeyMap(cfg)
	if km.Search.Keys()[0] != "ctrl+f" {
		t.Errorf("expected Search='ctrl+f', got %v", km.Search.Keys())
	}
	if km.Command.Keys()[0] != "ctrl+p" {
		t.Errorf("expected Command='ctrl+p', got %v", km.Command.Keys())
	}
}

func TestNewKeyMap_OverrideStarsKeys(t *testing.T) {
	cfg := &config.TUIKeybindings{
		Stars: config.StarsKeybindings{
			ToggleStar: "ctrl+s",
			EditCat:    "ctrl+c",
			EditTag:    "ctrl+t",
		},
	}
	km := NewKeyMap(cfg)
	if km.ToggleStar.Keys()[0] != "ctrl+s" {
		t.Errorf("expected ToggleStar='ctrl+s', got %v", km.ToggleStar.Keys())
	}
	if km.EditCategory.Keys()[0] != "ctrl+c" {
		t.Errorf("expected EditCategory='ctrl+c', got %v", km.EditCategory.Keys())
	}
	if km.EditTag.Keys()[0] != "ctrl+t" {
		t.Errorf("expected EditTag='ctrl+t', got %v", km.EditTag.Keys())
	}
}

func TestAllKeysNonEmpty(t *testing.T) {
	km := DefaultKeyMap()
	bindings := []struct {
		name    string
		binding interface{ Keys() []string }
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"FirstLine", km.FirstLine},
		{"LastLine", km.LastLine},
		{"NextSection", km.NextSection},
		{"PrevSection", km.PrevSection},
		{"NextGroup", km.NextGroup},
		{"PrevGroup", km.PrevGroup},
		{"NextView", km.NextView},
		{"PrevView", km.PrevView},
		{"ToggleSidebar", km.ToggleSidebar},
		{"Quit", km.Quit},
		{"Help", km.Help},
		{"OpenGithub", km.OpenGithub},
		{"Refresh", km.Refresh},
		{"Sync", km.Sync},
		{"Search", km.Search},
		{"Command", km.Command},
		{"Escape", km.Escape},
		{"Enter", km.Enter},
		{"ToggleStar", km.ToggleStar},
		{"EditCategory", km.EditCategory},
		{"EditTag", km.EditTag},
		{"Analyze", km.Analyze},
	}
	for _, b := range bindings {
		keys := b.binding.Keys()
		if len(keys) == 0 || strings.TrimSpace(keys[0]) == "" {
			t.Errorf("%s has no keys configured", b.name)
		}
	}
}
