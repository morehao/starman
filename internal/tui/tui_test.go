package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/morehao/starman/internal/tui/components"
	"github.com/morehao/starman/internal/tui/pages"
	"github.com/morehao/starman/internal/tui/styles"
)

func createTestModel(page PageID) *TuiModel {
	theme := styles.DefaultTheme()
	return &TuiModel{
		theme:       theme,
		currentPage: page,
		sidebar:     components.NewSidebar(theme),
		statusbar:   components.NewStatusBar(theme),
		taskCenter:  NewTaskCenter(),
		palette:     components.NewCommandPalette(theme, components.DefaultCommandCatalog()),
		workspace:   components.NewCommandWorkspace(theme),
		uiState:     StateNormal,
		focusPane:   FocusSidebar,
		pages: map[PageID]tea.Model{
			PageSearch: pages.NewSearch(nil, theme, nil),
		},
	}
}

func TestSearchPageLetterKeysStayOnSearchPage(t *testing.T) {
	// 搜索页面按字母键（包括之前被 ShortcutPage 拦截的 s/a/r/g/c/t/b/1）
	// 不应跳转到其他页面
	letters := []string{"s", "a", "r", "g", "c", "t", "b", "1", "z", "x", "m", "n"}

	for _, key := range letters {
		t.Run("key_"+key, func(t *testing.T) {
			m := createTestModel(PageSearch)
			model, _ := m.Update(keyMsg(key))

			tm := model.(*TuiModel)
			if tm.currentPage != PageSearch {
				t.Fatalf("key %q navigated away to page %v", key, tm.currentPage)
			}
		})
	}
}

func TestSearchPageSpecialKeysStayOnSearchPage(t *testing.T) {
	// 搜索页面按 tab/esc/up/down/j/k/enter 不应触发 sidebar 或全局导航
	keys := []string{"tab", "esc", "up", "down", "j", "k", "enter"}

	for _, key := range keys {
		t.Run("key_"+key, func(t *testing.T) {
			m := createTestModel(PageSearch)
			model, _ := m.Update(keyMsg(key))

			tm := model.(*TuiModel)
			if tm.currentPage != PageSearch {
				t.Fatalf("key %q navigated away to page %v", key, tm.currentPage)
			}
		})
	}
}

func TestSearchPageGlobalQuitStillWorks(t *testing.T) {
	// q 键在搜索页面仍能触发退出
	m := createTestModel(PageSearch)
	_, cmd := m.Update(keyMsg("q"))

	if cmd == nil {
		t.Fatal("expected quit command, got nil")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestSearchPageLettersReachTextInput(t *testing.T) {
	// 验证字母键确实进入了搜索框 textinput
	m := createTestModel(PageSearch)

	// 初始化搜索页面（Focus textinput）
	sp := m.pages[PageSearch].(*pages.SearchModel)
	sp.Init()

	// 输入 "h"
	model1, _ := m.Update(keyMsg("h"))
	tm1 := model1.(*TuiModel)
	sp1 := tm1.pages[PageSearch].(*pages.SearchModel)
	if sp1.InputValue() != "h" {
		t.Fatalf("expected input 'h', got %q", sp1.InputValue())
	}

	// 继续输入 "i"
	model2, _ := tm1.Update(keyMsg("i"))
	tm2 := model2.(*TuiModel)
	sp2 := tm2.pages[PageSearch].(*pages.SearchModel)
	if sp2.InputValue() != "hi" {
		t.Fatalf("expected input 'hi', got %q", sp2.InputValue())
	}
}

func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
