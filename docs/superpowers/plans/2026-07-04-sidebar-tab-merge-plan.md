# Sidebar Tab 合并实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Stars/Trending 视图 sidebar 从三 tab 切换改为单一滚动详情面板

**Architecture:** 重命名 `repoview` → `repodetail`，删除 tab 状态机，将 Overview + AI Summary（glamour 渲染）+ Release 订阅状态合并到一个 `renderDetail()` 输出。`ui.go` 中删除 `PrevSection`/`NextSection` 对 repoview tab 的切换调用。

**Tech Stack:** Go 1.22+, charm.land/bubbletea/v2, charm.land/lipgloss/v2, charmbracelet/glamour, modernc.org/sqlite

## Global Constraints

- 禁止在 `main` 分支上直接开发，所有改动在 `feat/merge-sidebar-tabs` 分支
- 颜色必须走 `theme.Theme` token，不允许硬编码
- 长耗时操作必须走 Task 系统，Update() 内禁止阻塞
- Stars 顶部 section tabs（All/Language/Category/Tag）不受影响
- Trending + Stats 视图不受影响（Stats 有自己的 tab 逻辑）
- 测试覆盖 L1+L2+L3，每层都有用例

---

### Task 1: 创建 repodetail 组件（新建文件）

**Files:**
- Create: `internal/tui/components/repodetail/repodetail.go`
- Create: `internal/tui/components/repodetail/repodetail_test.go`

**Interfaces:**
- Produces: `type Model struct` with `SetRepo(*store.Repository)`, `SetWidth(int)`, `Repo() *store.Repository`, `View() string`

- [ ] **Step 1: 创建 repodetail.go**

```go
package repodetail

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"

	"github.com/morehao/starman/internal/store"
)

type Model struct {
	repo  *store.Repository
	width int
}

func NewModel() *Model {
	return &Model{}
}

func (m *Model) SetRepo(r *store.Repository) { m.repo = r }

func (m *Model) SetWidth(w int) { m.width = w }

func (m Model) Repo() *store.Repository { return m.repo }

func (m Model) View() string {
	if m.repo == nil {
		return ""
	}
	return m.renderDetail()
}

func (m Model) renderDetail() string {
	var b strings.Builder

	repoName := lipgloss.NewStyle().Bold(true).Render(m.repo.FullName)
	meta := fmt.Sprintf("  \u2B50%d  \U0001F374%d", m.repo.StargazersCount, m.repo.ForksCount)
	b.WriteString(repoName + meta)

	if m.repo.Language != "" {
		b.WriteString("  " + m.repo.Language)
	}
	b.WriteString("\n")

	b.WriteString(strings.Repeat("\u2500", 50))
	b.WriteString("\n")

	if m.repo.Description != "" {
		b.WriteString("\n" + m.repo.Description)
	}

	fieldLabel := func(s string) string {
		return lipgloss.NewStyle().Bold(true).Width(10).Render(s)
	}

	if m.repo.Language != "" {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Language"), m.repo.Language))
	}

	if m.repo.AICategory != "" || m.repo.CustomCategory != "" {
		cat := m.repo.AICategory
		if cat == "" {
			cat = m.repo.CustomCategory
		}
		lock := ""
		if m.repo.CategoryLocked {
			lock = " \U0001F512"
		}
		b.WriteString(fmt.Sprintf("\n%s  %s%s", fieldLabel("Category"), cat, lock))
	}

	if len(m.repo.AIPlatforms) > 0 {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Platform"), strings.Join(m.repo.AIPlatforms, ", ")))
	}

	if m.repo.StarredAt != "" {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Starred"), m.repo.StarredAt))
	}

	if len(m.repo.Topics) > 0 {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Topics"), strings.Join(m.repo.Topics, ", ")))
	}

	allTags := append([]string{}, m.repo.AITags...)
	allTags = append(allTags, m.repo.CustomTags...)
	if len(allTags) > 0 {
		b.WriteString(fmt.Sprintf("\n%s  %s", fieldLabel("Tags"), strings.Join(allTags, ", ")))
	}

	if m.repo.AISummary != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("\u25B6 AI Summary"))
		b.WriteString("\n")
		b.WriteString(m.renderMarkdown(m.repo.AISummary))
	}

	if m.repo.Homepage != "" {
		b.WriteString(fmt.Sprintf("\n\n%s  %s", fieldLabel("Homepage"), m.repo.Homepage))
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("\u2500", 50))
	b.WriteString("\n")

	if m.repo.SubscribedReleases {
		b.WriteString(fmt.Sprintf("Release: Subscribed  |  Last fetched: %v", m.repo.LastReleaseFetch))
	} else {
		b.WriteString("Release: Not subscribed")
	}

	return b.String()
}

func (m Model) renderMarkdown(content string) string {
	renderWidth := m.width - 4
	if renderWidth < 20 {
		renderWidth = 20
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return rendered
}
```

- [ ] **Step 2: 创建 repodetail_test.go**

```go
package repodetail

import (
	"strings"
	"testing"

	"github.com/morehao/starman/internal/store"
)

func TestViewEmpty(t *testing.T) {
	m := NewModel()
	out := m.View()
	if out != "" {
		t.Fatalf("expected empty for nil repo, got: %q", out)
	}
}

func TestViewWithRepo(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:        "morehao/starman",
		StargazersCount: 128,
		ForksCount:      12,
		Language:        "Go",
		AICategory:      "开发工具",
		StarredAt:       "2025-06-01",
		Topics:          []string{"tui", "cli"},
		AISummary:       "A terminal UI framework",
	})
	out := m.View()

	checks := []string{"morehao/starman", "128", "12", "Go", "开发工具", "tui", "cli", "A terminal UI framework"}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output: %q", check, out)
		}
	}
}

func TestRendersAllFields(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:        "owner/repo",
		StargazersCount: 100,
		ForksCount:      5,
		Language:        "Rust",
		AICategory:      "库",
		CategoryLocked:  true,
		AIPlatforms:     []string{"cli"},
		StarredAt:       "2025-01-15",
		Topics:          []string{"parser"},
		AITags:          []string{"regex"},
		CustomTags:      []string{"favorite"},
		AISummary:       "Fast regex parser",
		Homepage:        "https://example.com",
	})
	out := m.View()

	checks := []string{
		"owner/repo", "100", "5", "Rust", "库", "cli",
		"2025-01-15", "parser", "regex", "favorite",
		"Fast regex parser", "https://example.com",
	}
	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Fatalf("missing %q in output: %q", check, out)
		}
	}
}

func TestAISummaryEmpty(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName: "noai/repo",
	})
	out := m.View()
	if strings.Contains(out, "AI Summary") {
		t.Fatalf("should not render AI Summary when empty: %q", out)
	}
}

func TestAISummaryGlamour(t *testing.T) {
	m := NewModel()
	m.SetWidth(80)
	m.SetRepo(&store.Repository{
		FullName:      "test/repo",
		AISummary:     "# Hello\n\nThis is a **test**.\n\n- item 1\n- item 2",
	})
	out := m.View()
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected glamour rendered content: %q", out)
	}
}

func TestAISummaryGlamourNarrow(t *testing.T) {
	m := NewModel()
	m.SetWidth(16)
	m.SetRepo(&store.Repository{
		FullName:      "test/repo",
		AISummary:     "# Hello\n\nSome markdown content",
	})
	out := m.View()
	if !strings.Contains(out, "Hello") {
		t.Fatalf("expected content in narrow render: %q", out)
	}
}

func TestReleaseSubscribed(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{
		FullName:           "owner/repo",
		SubscribedReleases: true,
	})
	out := m.View()
	if !strings.Contains(out, "Subscribed") {
		t.Fatalf("missing subscription status: %q", out)
	}
}

func TestReleaseNotSubscribed(t *testing.T) {
	m := NewModel()
	m.SetRepo(&store.Repository{FullName: "owner/repo"})
	out := m.View()
	if !strings.Contains(out, "Not subscribed") {
		t.Fatalf("missing not subscribed: %q", out)
	}
}

func TestNilRepo(t *testing.T) {
	m := NewModel()
	m.SetWidth(80)
	out := m.View()
	if out != "" {
		t.Fatalf("expected empty string for nil repo, got: %q", out)
	}
}
```

- [ ] **Step 3: 运行测试确认通过**

```bash
go test ./internal/tui/components/repodetail/ -v
```

Expected: 8 tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/components/repodetail/
git commit -m "feat: add repodetail component with merged sidebar tabs"
```

---

### Task 2: 更新 ui.go 引用，从 repoview 迁移到 repodetail

**Files:**
- Modify: `internal/tui/ui.go`

**Interfaces:**
- Consumes: `*repodetail.Model` (SetRepo, SetWidth, Repo, View)
- Produces: same as before, no interface change for consumers of `syncSidebar()`

- [ ] **Step 1: 更新 import 和类型引用**

在 `internal/tui/ui.go` 中：

将第 31 行的 import:
```go
"github.com/morehao/starman/internal/tui/components/repoview"
```
改为:
```go
"github.com/morehao/starman/internal/tui/components/repodetail"
```

将第 59 行的字段声明:
```go
repo        *repoview.Model
```
改为:
```go
repo        *repodetail.Model
```

将第 102 行的初始化:
```go
repo:        repoview.NewModel(),
```
改为:
```go
repo:        repodetail.NewModel(),
```

将第 354-367 行的 PrevSection/NextSection 处理:
```go
case key.Matches(typed, m.ctx.Keys.PrevSection):
	if m.ctx.View == tuicontext.StatsView {
		m.stats.PrevTab()
	} else {
		m.repo.PrevTab()
		m.syncSidebar()
	}
case key.Matches(typed, m.ctx.Keys.NextSection):
	if m.ctx.View == tuicontext.StatsView {
		m.stats.NextTab()
	} else {
		m.repo.NextTab()
		m.syncSidebar()
	}
```
改为:
```go
case key.Matches(typed, m.ctx.Keys.PrevSection):
	if m.ctx.View == tuicontext.StatsView {
		m.stats.PrevTab()
	}
case key.Matches(typed, m.ctx.Keys.NextSection):
	if m.ctx.View == tuicontext.StatsView {
		m.stats.NextTab()
	}
```

- [ ] **Step 2: 运行测试确认 ui.go 编译通过**

```bash
go build ./internal/tui/
```

Expected: no errors

- [ ] **Step 3: 运行全量测试**

```bash
go test ./internal/tui/...
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/ui.go
git commit -m "refactor: migrate ui.go from repoview to repodetail"
```

---

### Task 3: 删除旧 repoview 目录

**Files:**
- Delete: `internal/tui/components/repoview/repoview.go`
- Delete: `internal/tui/components/repoview/repoview_test.go`

- [ ] **Step 1: 确认无其他文件引用 repoview**

```bash
rg "repoview" --include "*.go" -l
```

Expected: no results (only the repoview directory itself)

- [ ] **Step 2: 删除旧文件**

```bash
rm -rf internal/tui/components/repoview/
```

- [ ] **Step 3: 全量测试**

```bash
go test ./...
```

Expected: all tests PASS

- [ ] **Step 4: 全量 lint**

```bash
make lint
```

Expected: no new warnings

- [ ] **Step 5: Commit**

```bash
git add internal/tui/components/repoview/
git commit -m "refactor: remove old repoview component"
```

---

### Task 4: 更新 footer help 文本和 keys 描述

**Files:**
- Modify: `internal/tui/components/footer/footer.go`
- Modify: `internal/tui/keys/keys.go`

- [ ] **Step 1: 更新 footer help 文本**

在 `internal/tui/components/footer/footer.go:46`：

```go
helpText := "j/k move  g/G first/last  h/l tab  p sidebar  / search  : cmd  q quit"
```
改为:
```go
helpText := "j/k move  g/G first/last  p sidebar  / search  : cmd  q quit"
```

- [ ] **Step 2: 可选 — 将 PrevSection/NextSection 标记为 Deprecated（保留在 KeyMap 中保持兼容）**

在 `internal/tui/keys/keys.go` 第 14-15 行的 `NextSection`/`PrevSection` 字段上方添加注释：

```go
// Deprecated: Sidebar tabs have been merged. Only used by Stats view.
NextSection   key.Binding
// Deprecated: Sidebar tabs have been merged. Only used by Stats view.
PrevSection   key.Binding
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/tui/... -v
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/tui/components/footer/footer.go internal/tui/keys/keys.go
git commit -m "refactor: update footer help text for merged sidebar tabs"
```

---

### Task 5: 最终验证

- [ ] **Step 1: 全量测试**

```bash
go test ./...
```

- [ ] **Step 2: 全量 lint**

```bash
make lint
```

- [ ] **Step 3: 编译**

```bash
make build
```

- [ ] **Step 4: 确认无 repoview 残留引用**

```bash
rg "repoview" --include "*.go"
```

Expected: no results

- [ ] **Step 5: 确认 golden 测试仍通过（如有）**

```bash
go test ./internal/tui/... -v -count=1
```

- [ ] **Step 6: 最终 Commit（如有遗漏改动）**
