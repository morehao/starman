package prompt

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/morehao/starman/internal/tui/theme"
)

func testModel() Model {
	return Model{th: theme.DefaultTheme(), width: 80, active: true}
}

func TestNewConfirmModel(t *testing.T) {
	m := NewConfirmModel("Delete?")
	if m.ptype != PromptConfirm {
		t.Errorf("expected PromptConfirm, got %v", m.ptype)
	}
	if !m.active {
		t.Error("expected active=true")
	}
	if m.title != "Delete?" {
		t.Errorf("expected 'Delete?', got %q", m.title)
	}
}

func TestNewCategorySelectModel(t *testing.T) {
	cats := []string{"web", "cli", "ai"}
	m := NewCategorySelectModel("Pick", cats, "cli")
	if m.ptype != PromptCategorySelect {
		t.Errorf("expected PromptCategorySelect, got %v", m.ptype)
	}
	if m.cursor != 1 {
		t.Errorf("expected cursor=1 for 'cli', got %d", m.cursor)
	}
}

func TestNewCategorySelectModel_NoCurrent(t *testing.T) {
	cats := []string{"web", "cli"}
	m := NewCategorySelectModel("Pick", cats, "")
	if m.cursor != 0 {
		t.Errorf("expected cursor=0 for empty current, got %d", m.cursor)
	}
}

func TestNewTagEditModel(t *testing.T) {
	m := NewTagEditModel("Edit tag", "current-tag")
	if m.ptype != PromptTagEdit {
		t.Errorf("expected PromptTagEdit, got %v", m.ptype)
	}
	if m.input != "current-tag" {
		t.Errorf("expected 'current-tag', got %q", m.input)
	}
}

func TestNewCategoryFormModel(t *testing.T) {
	fields := []FormField{
		NewFormField("ID", "", false, false),
		NewFormField("Name", "", false, false),
	}
	m := NewCategoryFormModel("New Category", fields, false)
	if m.ptype != PromptCategoryForm {
		t.Errorf("expected PromptCategoryForm, got %v", m.ptype)
	}
	if len(m.formFields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(m.formFields))
	}
	if m.formFieldIdx != 0 {
		t.Errorf("expected formFieldIdx=0, got %d", m.formFieldIdx)
	}
}

func TestNewFormField(t *testing.T) {
	f := NewFormField("Keywords", "go,web", false, true)
	if f.Label != "Keywords" {
		t.Errorf("expected 'Keywords', got %q", f.Label)
	}
	if f.Value != "go,web" {
		t.Errorf("expected 'go,web', got %q", f.Value)
	}
	if f.Readonly != true {
		t.Errorf("expected readonly=true")
	}
}

func TestIsFocused(t *testing.T) {
	m := NewConfirmModel("test")
	if !m.IsFocused() {
		t.Error("expected focused when active")
	}
}

func TestSetTheme(t *testing.T) {
	m := NewConfirmModel("test")
	m.SetTheme(theme.LightTheme())
	if m.th != (theme.LightTheme()) {
		t.Error("theme not set correctly")
	}
}

func TestConfirm_Escape(t *testing.T) {
	m := testModel()
	m.ptype = PromptConfirm
	m.active = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 27})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after escape")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	pr, ok := msg.(PromptResultMsg)
	if !ok {
		t.Fatalf("expected PromptResultMsg, got %T", msg)
	}
	if pr.Confirmed {
		t.Error("expected Confirmed=false on escape")
	}
}

func TestConfirm_Yes(t *testing.T) {
	m := testModel()
	m.ptype = PromptConfirm
	m.active = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after y")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if !pr.Confirmed {
		t.Error("expected Confirmed=true on y")
	}
}

func TestConfirm_No(t *testing.T) {
	m := testModel()
	m.ptype = PromptConfirm
	m.active = true
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after n")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if pr.Confirmed {
		t.Error("expected Confirmed=false on n")
	}
}

func TestCategorySelect_Escape(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategorySelect
	m.active = true
	m.options = []string{"web", "cli"}
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 27})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after escape")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if pr.Confirmed {
		t.Error("expected Confirmed=false on escape")
	}
}

func TestCategorySelect_Enter(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategorySelect
	m.active = true
	m.options = []string{"web", "cli"}
	m.cursor = 1
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 13})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after enter")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if !pr.Confirmed {
		t.Error("expected Confirmed=true on enter")
	}
	if pr.Value != "cli" {
		t.Errorf("expected 'cli', got %q", pr.Value)
	}
}

func TestCategorySelect_UpNavigation(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategorySelect
	m.options = []string{"web", "cli", "ai"}
	m.cursor = 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	nm := updated.(Model)
	if nm.cursor != 0 {
		t.Errorf("expected cursor=0 after up, got %d", nm.cursor)
	}
}

func TestCategorySelect_UpAtTopStays(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategorySelect
	m.options = []string{"web", "cli"}
	m.cursor = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	nm := updated.(Model)
	if nm.cursor != 0 {
		t.Errorf("expected cursor=0 when at top, got %d", nm.cursor)
	}
}

func TestTagEdit_Escape(t *testing.T) {
	m := testModel()
	m.ptype = PromptTagEdit
	m.active = true
	m.input = "test"
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 27})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after escape")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
}

func TestTagEdit_Enter(t *testing.T) {
	m := testModel()
	m.ptype = PromptTagEdit
	m.active = true
	m.input = "mytag"
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 13})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after enter")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if pr.Value != "mytag" {
		t.Errorf("expected 'mytag', got %q", pr.Value)
	}
}

func TestTagEdit_CharInput(t *testing.T) {
	m := testModel()
	m.ptype = PromptTagEdit
	m.active = true

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	nm := updated.(Model)
	if nm.input != "k" {
		t.Errorf("expected 'k', got %q", nm.input)
	}
	// Cursor should advance
	if nm.cursorPos != 1 {
		t.Errorf("expected cursorPos=1 after typing 'k', got %d", nm.cursorPos)
	}
}

func TestTagEdit_Backspace(t *testing.T) {
	m := testModel()
	m.ptype = PromptTagEdit
	m.active = true
	m.input = "ab"
	m.cursorPos = len(m.input)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 127})
	nm := updated.(Model)
	if nm.input != "a" {
		t.Errorf("expected 'a', got %q", nm.input)
	}
	if nm.cursorPos != 1 {
		t.Errorf("expected cursorPos=1 after backspace, got %d", nm.cursorPos)
	}
}

func TestTagEdit_LeftRightMovesCursor(t *testing.T) {
	m := testModel()
	m.ptype = PromptTagEdit
	m.active = true
	m.input = "abc"
	m.cursorPos = len(m.input)

	m1, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m1.(Model).cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after Left, got %d", m1.(Model).cursorPos)
	}

	m2, _ := m1.(Model).Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m2.(Model).cursorPos != 3 {
		t.Errorf("expected cursorPos=3 after Right, got %d", m2.(Model).cursorPos)
	}
}

func TestTagEdit_InsertAtCursor(t *testing.T) {
	m := testModel()
	m.ptype = PromptTagEdit
	m.active = true
	m.input = "ab"
	m.cursorPos = 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'X'})
	nm := updated.(Model)
	if nm.input != "aXb" {
		t.Errorf("expected 'aXb' after insert at pos 1, got %q", nm.input)
	}
	if nm.cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after insert, got %d", nm.cursorPos)
	}
}

// --- CategoryForm tests (the fix) ---

func testCategoryFormModel() Model {
	m := testModel()
	m.ptype = PromptCategoryForm
	m.active = true
	m.formFields = []FormField{
		{Label: "ID", Value: "", Readonly: false},
		{Label: "Name", Value: "", Readonly: false},
		{Label: "Keywords", Value: "", Readonly: false},
	}
	m.formFieldIdx = 0
	return m
}

func TestCategoryForm_KKeyTypesCharacter(t *testing.T) {
	m := testCategoryFormModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	nm := updated.(Model)
	if nm.formFields[0].Value != "k" {
		t.Errorf("expected 'k' typed into field 0, got %q", nm.formFields[0].Value)
	}
	if nm.formFieldIdx != 0 {
		t.Errorf("formFieldIdx should remain 0, got %d", nm.formFieldIdx)
	}
}

func TestCategoryForm_JKeyTypesCharacter(t *testing.T) {
	m := testCategoryFormModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	nm := updated.(Model)
	if nm.formFields[0].Value != "j" {
		t.Errorf("expected 'j' typed into field 0, got %q", nm.formFields[0].Value)
	}
	if nm.formFieldIdx != 0 {
		t.Errorf("formFieldIdx should remain 0, got %d", nm.formFieldIdx)
	}
}

func TestCategoryForm_TypingSkills(t *testing.T) {
	m := testCategoryFormModel()

	for _, r := range "skills" {
		updated, _ := m.Update(tea.KeyPressMsg{Code: r})
		m = updated.(Model)
	}

	if m.formFields[0].Value != "skills" {
		t.Errorf("expected 'skills', got %q", m.formFields[0].Value)
	}
	if m.formFieldIdx != 0 {
		t.Errorf("field should not change during typing, got idx=%d", m.formFieldIdx)
	}
}

func TestCategoryForm_NavigateFields(t *testing.T) {
	m := testCategoryFormModel()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	nm := updated.(Model)
	if nm.formFieldIdx != 1 {
		t.Errorf("expected formFieldIdx=1 after down, got %d", nm.formFieldIdx)
	}

	updated, _ = nm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	nm = updated.(Model)
	if nm.formFieldIdx != 2 {
		t.Errorf("expected formFieldIdx=2 after second down, got %d", nm.formFieldIdx)
	}
}

func TestCategoryForm_NavigateFieldsWraps(t *testing.T) {
	m := testCategoryFormModel()
	m.formFieldIdx = 2

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	nm := updated.(Model)
	if nm.formFieldIdx != 0 {
		t.Errorf("expected formFieldIdx=0 after wrap, got %d", nm.formFieldIdx)
	}
}

func TestCategoryForm_NavigateUp(t *testing.T) {
	m := testCategoryFormModel()
	m.formFieldIdx = 2

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	nm := updated.(Model)
	if nm.formFieldIdx != 1 {
		t.Errorf("expected formFieldIdx=1 after up, got %d", nm.formFieldIdx)
	}
}

func TestCategoryForm_NavigateUpWraps(t *testing.T) {
	m := testCategoryFormModel()
	m.formFieldIdx = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	nm := updated.(Model)
	if nm.formFieldIdx != 2 {
		t.Errorf("expected formFieldIdx=2 after wrap, got %d", nm.formFieldIdx)
	}
}

func TestCategoryForm_TypeInDifferentField(t *testing.T) {
	m := testCategoryFormModel()
	m.formFieldIdx = 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	nm := updated.(Model)
	if nm.formFields[1].Value != "k" {
		t.Errorf("expected 'k' in field 1, got %q", nm.formFields[1].Value)
	}
	if nm.formFields[0].Value != "" {
		t.Errorf("field 0 should be empty, got %q", nm.formFields[0].Value)
	}
}

func TestCategoryForm_Backspace(t *testing.T) {
	m := testCategoryFormModel()
	m.formFields[0].Value = "ab"
	m.cursorPos = len(m.formFields[0].Value)

	updated, _ := m.Update(tea.KeyPressMsg{Code: 127})
	nm := updated.(Model)
	if nm.formFields[0].Value != "a" {
		t.Errorf("expected 'a', got %q", nm.formFields[0].Value)
	}
	if nm.cursorPos != 1 {
		t.Errorf("expected cursorPos=1 after backspace, got %d", nm.cursorPos)
	}
}

func TestCategoryForm_LeftRightMovesCursor(t *testing.T) {
	m := testCategoryFormModel()
	m.formFields[0].Value = "abc"
	m.cursorPos = len(m.formFields[0].Value)

	m1, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	nm := m1.(Model)
	if nm.cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after Left, got %d", nm.cursorPos)
	}

	m2, _ := nm.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m2.(Model).cursorPos != 3 {
		t.Errorf("expected cursorPos=3 after Right, got %d", m2.(Model).cursorPos)
	}
}

func TestCategoryForm_NavigateFieldResetsCursor(t *testing.T) {
	m := testCategoryFormModel()
	m.formFields[0].Value = "abc"
	m.formFields[1].Value = "xy"
	m.cursorPos = 1

	// Move down, cursor should reset to end of field 1's value
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	nm := updated.(Model)
	if nm.formFieldIdx != 1 {
		t.Errorf("expected formFieldIdx=1, got %d", nm.formFieldIdx)
	}
	if nm.cursorPos != 2 {
		t.Errorf("expected cursorPos=2 (len of 'xy'), got %d", nm.cursorPos)
	}
}

func TestCategoryForm_InsertAtCursor(t *testing.T) {
	m := testCategoryFormModel()
	m.formFields[0].Value = "ab"
	m.cursorPos = 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'X'})
	nm := updated.(Model)
	if nm.formFields[0].Value != "aXb" {
		t.Errorf("expected 'aXb' after insert at pos 1, got %q", nm.formFields[0].Value)
	}
	if nm.cursorPos != 2 {
		t.Errorf("expected cursorPos=2 after insert, got %d", nm.cursorPos)
	}
}

func TestCategoryForm_EnterSubmits(t *testing.T) {
	m := testCategoryFormModel()
	m.formFields[0].Value = "my-cat"
	m.formFields[1].Value = "My Category"

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 13})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after enter")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if !pr.Confirmed {
		t.Error("expected Confirmed=true on enter")
	}
	parts := pr.Value
	if parts != "my-cat\x00My Category\x00" {
		t.Errorf("unexpected value: %q", parts)
	}
}

func TestCategoryForm_EscapeCancels(t *testing.T) {
	m := testCategoryFormModel()

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 27})
	nm := updated.(Model)
	if nm.active {
		t.Error("expected active=false after escape")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	pr := msg.(PromptResultMsg)
	if pr.Confirmed {
		t.Error("expected Confirmed=false on escape")
	}
}

func TestCategoryForm_BoolToggle(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategoryForm
	m.active = true
	m.formFields = []FormField{
		{Label: "IsHidden", Value: "false", IsBool: true},
	}
	m.formFieldIdx = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	nm := updated.(Model)
	if nm.formFields[0].Value != "true" {
		t.Errorf("expected 'true' after space, got %q", nm.formFields[0].Value)
	}
}

func TestCategoryForm_ReadonlyFieldIgnored(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategoryForm
	m.active = true
	m.formFields = []FormField{
		{Label: "ID", Value: "fixed", Readonly: true},
	}
	m.formFieldIdx = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	nm := updated.(Model)
	if nm.formFields[0].Value != "fixed" {
		t.Errorf("readonly field should not change, got %q", nm.formFields[0].Value)
	}
}

func TestCategoryForm_SkipsBoolFieldsForCharInput(t *testing.T) {
	m := testModel()
	m.ptype = PromptCategoryForm
	m.active = true
	m.formFields = []FormField{
		{Label: "IsHidden", Value: "false", IsBool: true},
	}
	m.formFieldIdx = 0

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	nm := updated.(Model)
	if nm.formFields[0].Value != "false" {
		t.Errorf("bool field should not receive char input, got %q", nm.formFields[0].Value)
	}
}

func TestInactiveModelIgnoresKeys(t *testing.T) {
	m := testModel()
	m.active = false
	m.ptype = PromptConfirm

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
	nm := updated.(Model)
	if cmd != nil {
		t.Error("inactive model should not produce cmd")
	}
	if nm.active {
		t.Error("expected inactive")
	}
}

func TestView_Confirm(t *testing.T) {
	m := NewConfirmModel("Delete?")
	m.th = theme.DefaultTheme()
	m.width = 80
	v := m.View()
	if v.Content == "" {
		t.Error("expected non-empty view")
	}
}

func TestView_Inactive(t *testing.T) {
	m := NewConfirmModel("test")
	m.active = false
	v := m.View()
	if v.Content != "" {
		t.Errorf("expected empty view when inactive, got %q", v.Content)
	}
}
