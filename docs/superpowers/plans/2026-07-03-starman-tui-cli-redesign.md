# starman TUI/CLI 重构 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 starman 收敛为 TUI 主入口，CLI 仅保留 `config/completion/sync/search`，并让 TUI 关键页面接入真实能力与统一任务交互。

**Architecture:** 新增 `internal/app` 统一 Action 层，CLI 与 TUI 共用同一执行路径；TUI 引入 TaskCenter 管理异步任务生命周期；导航与页面交互按任务流重构。所有页面只负责状态与渲染，不直接拼接业务逻辑。

**Tech Stack:** Go 1.25、cobra、bubbletea/lipgloss、现有 internal 业务包（store/github/ai/generate/release/backup/discovery）

## Global Constraints

- 迁移策略允许破坏性调整（项目仍在开发阶段）。
- CLI 保留集合为: `config`、`completion`、`sync`、`search`、`--help` / `--version`。
- TUI 交互采用混合执行模型：默认异步后台执行；关键高风险操作前台确认。
- 一级导航按任务流分组。
- 布局最小设计基线为 `100x30`。

---

## File Structure

- Create: `internal/app/result.go`（统一 Action 输出模型）
- Create: `internal/app/sync_action.go`（sync 统一执行）
- Create: `internal/app/search_action.go`（search 统一执行）
- Create: `internal/app/sync_action_test.go`（sync action 单测）
- Create: `internal/app/search_action_test.go`（search action 单测）
- Create: `internal/cli/root_test.go`（命令树保留集回归）
- Modify: `internal/cli/root.go`（移除下沉命令，保留 CLI 集）
- Modify: `internal/cli/sync.go`（调用 app.SyncAction）
- Modify: `internal/cli/search.go`（调用 app.SearchAction）
- Create: `internal/tui/taskcenter.go`（任务队列与状态机）
- Create: `internal/tui/taskcenter_test.go`（任务状态流测试）
- Modify: `internal/tui/messages.go`（任务消息类型扩展）
- Modify: `internal/tui/tui.go`（接入 TaskCenter + 全局路由）
- Modify: `internal/tui/components/sidebar.go`（任务流分组导航）
- Modify: `internal/tui/components/sidebar_test.go`（分组导航与快捷键回归）
- Modify: `internal/tui/pages/sync.go`（移除假进度，改真实 action）
- Modify: `internal/tui/pages/search.go`（调用真实 search action）
- Modify: `internal/tui/pages/analyze.go`（接入真实分析 action）
- Modify: `internal/tui/pages/generate.go`（接入真实生成 action）
- Modify: `internal/tui/pages/release.go`（接入真实 release pull）
- Modify: `internal/tui/pages/backup.go`（导出异步 + 导入确认）
- Modify: `internal/tui/pages/trending.go`（接入 discovery service）
- Modify: `README.md`、`README.zh.md`（命令与交互说明更新）

---

### Task 1: 收敛 CLI 命令树并建立回归测试

**Files:**
- Create: `internal/cli/root_test.go`
- Modify: `internal/cli/root.go`
- Modify: `README.md`
- Modify: `README.zh.md`

**Interfaces:**
- Consumes: `NewRootCmd(ver string) *cobra.Command`
- Produces: root 子命令集合仅包含 `sync/search/config/completion`

- [ ] **Step 1: 先写失败测试约束 root 命令集合**

```go
package cli

import "testing"

func TestRootCommandSet(t *testing.T) {
	root := NewRootCmd("test")
	got := map[string]bool{}
	for _, c := range root.Commands() {
		got[c.Name()] = true
	}
	wantPresent := []string{"sync", "search", "config", "completion", "help"}
	for _, name := range wantPresent {
		if !got[name] {
			t.Fatalf("expected command %q to exist", name)
		}
	}
	wantAbsent := []string{"analyze", "tag", "categorize", "stats", "release", "generate", "backup", "info", "trending"}
	for _, name := range wantAbsent {
		if got[name] {
			t.Fatalf("expected command %q to be removed from root", name)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/cli -run TestRootCommandSet -v`
Expected: FAIL，提示仍存在被下沉命令。

- [ ] **Step 3: 修改 root.go，仅注册保留命令**

```go
root.AddCommand(newSyncCmd())
root.AddCommand(newSearchCmd())
root.AddCommand(newConfigCmd())
root.AddCommand(newCompletionCmd(root))
```

- [ ] **Step 4: 更新 README 命令用法段落**

```md
starman                    Launch the interactive TUI (default)
starman sync               Sync starred repositories
starman search <query>     Search repositories
starman config init|show   Manage config
starman completion <shell> Generate shell completion
```

- [ ] **Step 5: 回归并提交**

Run: `go test ./internal/cli -run TestRootCommandSet -v && go test ./...`
Expected: PASS

```bash
git add internal/cli/root.go internal/cli/root_test.go README.md README.zh.md
git commit -m "refactor(cli): keep minimal command set and update docs"
```

---

### Task 2: 新增 app Action 层（sync/search）与统一 Result

**Files:**
- Create: `internal/app/result.go`
- Create: `internal/app/sync_action.go`
- Create: `internal/app/search_action.go`
- Create: `internal/app/sync_action_test.go`
- Create: `internal/app/search_action_test.go`

**Interfaces:**
- Consumes: `store.Store`、`ai.Service`、`github.Client`
- Produces:
  - `type Result struct { Summary string; Warnings []string; Metrics map[string]string; Err error }`
  - `type SyncAction interface { Run(ctx context.Context, opts SyncOpts) (*SyncResult, error) }`
  - `type SearchAction interface { Run(ctx context.Context, query string, opts SearchOpts) (*SearchResult, error) }`

- [ ] **Step 1: 写 Result 与 action 测试（先失败）**

```go
func TestSyncActionRun(t *testing.T) {
	a := NewSyncAction(fakeStore, fakeGitHub)
	res, err := a.Run(context.Background(), SyncOpts{Full: false})
	if err != nil { t.Fatal(err) }
	if res.Fetched <= 0 { t.Fatalf("expected fetched > 0") }
}

func TestSearchActionRun(t *testing.T) {
	a := NewSearchAction(fakeStore, fakeAI)
	res, err := a.Run(context.Background(), "cli tool", SearchOpts{Limit: 5})
	if err != nil { t.Fatal(err) }
	if len(res.Hits) == 0 { t.Fatalf("expected at least one hit") }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/app -v`
Expected: FAIL（类型与构造函数未定义）。

- [ ] **Step 3: 实现 result.go 与 action 核心结构**

```go
type Result struct {
	Summary  string
	Warnings []string
	Metrics  map[string]string
	Err      error
}

type SyncOpts struct { Full bool; Watch bool; Interval time.Duration }
type SyncResult struct { Result; Fetched int; NewCount int }

type SearchOpts struct { Lang, Category, Sort string; Limit int }
type SearchResult struct { Result; Hits []*ai.SearchHit }
```

- [ ] **Step 4: 实现最小可用 Run 逻辑让测试通过**

```go
func (a *syncAction) Run(ctx context.Context, opts SyncOpts) (*SyncResult, error) {
	repos, err := a.gh.ListStarred(ctx, a.username)
	if err != nil { return nil, err }
	if err := a.store.UpsertReposOnSync(ctx, repos, opts.Full); err != nil { return nil, err }
	return &SyncResult{Fetched: len(repos)}, nil
}
```

- [ ] **Step 5: 回归并提交**

Run: `go test ./internal/app -v`
Expected: PASS

```bash
git add internal/app
git commit -m "feat(app): add shared result model and sync/search actions"
```

---

### Task 3: CLI `sync/search` 切换到 Action 层

**Files:**
- Modify: `internal/cli/sync.go`
- Modify: `internal/cli/search.go`
- Modify: `internal/cli/search_test.go`

**Interfaces:**
- Consumes: `app.NewSyncAction(...)`、`app.NewSearchAction(...)`
- Produces: CLI 与 TUI 共用统一 action 行为

- [ ] **Step 1: 写失败测试，断言 CLI 调用 action 输出路径**

```go
func TestSearchCommandUsesAction(t *testing.T) {
	cmd := newSearchCmd()
	if cmd == nil { t.Fatal("expected non-nil command") }
	if cmd.Flags().Lookup("limit") == nil { t.Fatal("expected --limit flag") }
}
```

- [ ] **Step 2: 运行测试确认基线**

Run: `go test ./internal/cli -run TestSearchCommandUsesAction -v`
Expected: 若失败则修复接口后继续。

- [ ] **Step 3: 改造 sync.go/search.go 调用 app action**

```go
action := app.NewSyncAction(s, gh, cfg.GitHub.Username)
res, err := action.Run(ctx, app.SyncOpts{Full: fullSync})
if err != nil { return err }
fmt.Fprintf(os.Stderr, "fetched=%d new=%d\n", res.Fetched, res.NewCount)
```

```go
action := app.NewSearchAction(s, svc)
res, err := action.Run(ctx, args[0], app.SearchOpts{Lang: lang, Category: category, Sort: sortBy, Limit: limit})
if err != nil { return err }
outputSearchTable(res.Hits)
```

- [ ] **Step 4: 回归 CLI 相关测试与 help 输出**

Run: `go test ./internal/cli -v && go build -o /tmp/starman ./cmd/starman && /tmp/starman sync --help && /tmp/starman search --help`
Expected: PASS 且命令帮助正常。

- [ ] **Step 5: 提交**

```bash
git add internal/cli/sync.go internal/cli/search.go internal/cli/search_test.go
git commit -m "refactor(cli): route sync and search through app actions"
```

---

### Task 4: 引入 TUI TaskCenter 与任务消息协议

**Files:**
- Create: `internal/tui/taskcenter.go`
- Create: `internal/tui/taskcenter_test.go`
- Modify: `internal/tui/messages.go`
- Modify: `internal/tui/tui.go`

**Interfaces:**
- Consumes: `tea.Msg` 分发机制
- Produces:
  - `type TaskState string` (`queued/running/success/failed/cancelled`)
  - `func (tc *TaskCenter) Enqueue(task Task) tea.Cmd`
  - `TaskStartedMsg/TaskProgressMsg/TaskDoneMsg`（统一结构）

- [ ] **Step 1: 写任务状态机失败测试**

```go
func TestTaskCenterLifecycle(t *testing.T) {
	tc := NewTaskCenter()
	id := tc.EnqueueMeta("sync")
	if tc.State(id) != TaskQueued { t.Fatalf("want queued") }
	tc.MarkRunning(id)
	tc.MarkDone(id, nil)
	if tc.State(id) != TaskSuccess { t.Fatalf("want success") }
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test ./internal/tui -run TestTaskCenterLifecycle -v`
Expected: FAIL（TaskCenter 未定义）。

- [ ] **Step 3: 实现 taskcenter.go 与 messages.go 扩展**

```go
type TaskState string
const (
	TaskQueued TaskState = "queued"
	TaskRunning TaskState = "running"
	TaskSuccess TaskState = "success"
	TaskFailed TaskState = "failed"
)
```

- [ ] **Step 4: 在 tui.go 中挂载并消费任务消息**

```go
case TaskStartedMsg:
	m.taskCenter.MarkRunning(msg.ID)
case TaskDoneMsg:
	m.taskCenter.MarkDone(msg.ID, msg.Err)
	m.statusbar.SetTaskSummary(m.taskCenter.Summary())
```

- [ ] **Step 5: 回归并提交**

Run: `go test ./internal/tui -v`
Expected: PASS

```bash
git add internal/tui/taskcenter.go internal/tui/taskcenter_test.go internal/tui/messages.go internal/tui/tui.go
git commit -m "feat(tui): add task center and unified task lifecycle messages"
```

---

### Task 5: 重构侧栏为任务流分组与统一导航交互

**Files:**
- Modify: `internal/tui/components/sidebar.go`
- Modify: `internal/tui/components/sidebar_test.go`
- Modify: `internal/tui/tui_test.go`

**Interfaces:**
- Consumes: `types.PageID`
- Produces: 任务流分组菜单 + 快捷键映射

- [ ] **Step 1: 写失败测试（分组标题与快捷键）**

```go
func TestSidebarGroups(t *testing.T) {
	m := NewSidebar(styles.DefaultTheme())
	v := m.View()
	for _, token := range []string{"发现", "整理", "处理", "系统"} {
		if !strings.Contains(v, token) { t.Fatalf("missing group %s", token) }
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tui/components -run TestSidebarGroups -v`
Expected: FAIL（当前未分组）。

- [ ] **Step 3: 修改 sidebar.go 使用分组结构**

```go
type menuGroup struct { title string; items []menuItem }
var groups = []menuGroup{
	{title: "发现", items: []menuItem{{"Search", "/", types.PageSearch}, {"Trending", "t", types.PageTrending}}},
	{title: "整理", items: []menuItem{{"Repo List", "r", types.PageRepoList}, {"Tag", "g", types.PageTag}}},
}
```

- [ ] **Step 4: 更新 tui 测试中的快捷键期望**

```go
shortcutTests := []struct{ key string; expected PageID }{
	{"/", PageSearch}, {"t", PageTrending}, {"r", PageRepoList}, {"s", PageSync},
}
```

- [ ] **Step 5: 回归并提交**

Run: `go test ./internal/tui/components ./internal/tui -v`
Expected: PASS

```bash
git add internal/tui/components/sidebar.go internal/tui/components/sidebar_test.go internal/tui/tui_test.go
git commit -m "refactor(tui): group sidebar navigation by task flow"
```

---

### Task 6: TUI 页面接线真实能力（先交付 Sync/Search/Trending）

**Files:**
- Modify: `internal/tui/pages/sync.go`
- Modify: `internal/tui/pages/search.go`
- Modify: `internal/tui/pages/trending.go`
- Modify: `internal/tui/tui.go`

**Interfaces:**
- Consumes: `app.SyncAction`、`app.SearchAction`、`discovery.Service`
- Produces: 3 个页面不再使用模拟逻辑

- [ ] **Step 1: 写失败测试，验证 Sync 页面不再返回假 step**

```go
func TestSyncPageNoFakeProgress(t *testing.T) {
	m := NewSync(nil, styles.DefaultTheme())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil { t.Fatalf("expected real task command") }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tui/pages -run TestSyncPageNoFakeProgress -v`
Expected: FAIL（仍为本地 stepSyncCmd）。

- [ ] **Step 3: 改造 sync/search/trending 页面调用 action/service**

```go
case "enter":
	return m, m.enqueueSyncTaskCmd(m.fullSync)
```

```go
func (m *SearchModel) doSearch() {
	res, err := m.searchAction.Run(context.Background(), q, app.SearchOpts{Limit: 50, Sort: "score"})
	if err != nil { m.err = err; return }
	m.results = extractRepos(res.Hits)
}
```

```go
repos, err := m.discovery.Trending(ctx, discovery.TrendingOpts{Since: m.since, Source: m.source, Top: 20})
```

- [ ] **Step 4: 页面联调回归**

Run: `go test ./internal/tui/... -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/tui/pages/sync.go internal/tui/pages/search.go internal/tui/pages/trending.go internal/tui/tui.go
git commit -m "feat(tui): wire sync/search/trending pages to real services"
```

---

### Task 7: TUI 页面接线真实能力（Analyze/Generate/Release/Backup）

**Files:**
- Modify: `internal/tui/pages/analyze.go`
- Modify: `internal/tui/pages/generate.go`
- Modify: `internal/tui/pages/release.go`
- Modify: `internal/tui/pages/backup.go`

**Interfaces:**
- Consumes: `ai.BatchAnalyzer`、`generate.Generator`、`release.Tracker`、`backup.ExportJSON/ImportJSON`
- Produces: 处理类页面全部走真实执行路径

- [ ] **Step 1: 写失败测试，覆盖 backup 导入确认与任务派发**

```go
func TestBackupImportRequiresConfirm(t *testing.T) {
	m := NewBackup(nil, styles.DefaultTheme())
	m.mode = "import"
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil { t.Fatalf("expected confirm step before task execution") }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/tui/pages -run TestBackupImportRequiresConfirm -v`
Expected: FAIL（当前未有确认门）。

- [ ] **Step 3: 接线真实任务并添加高风险确认门**

```go
if m.mode == "import" && !m.confirmed {
	m.confirmPrompt = "Import may overwrite local data. Press Enter again to confirm."
	m.confirmed = true
	return m, nil
}
return m, m.enqueueImportTaskCmd(m.importPath)
```

- [ ] **Step 4: 全量 TUI 回归**

Run: `go test ./internal/tui/... -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add internal/tui/pages/analyze.go internal/tui/pages/generate.go internal/tui/pages/release.go internal/tui/pages/backup.go
git commit -m "feat(tui): implement real processing actions with confirm gates"
```

---

### Task 8: 最终验收（命令面 + 交互面 + 功能面）

**Files:**
- Modify: `README.md`
- Modify: `README.zh.md`
- Modify: `internal/tui/tui_test.go`

**Interfaces:**
- Consumes: 全部前置任务产物
- Produces: 可验证的 DoD 结果

- [ ] **Step 1: 增加验收测试（命令树、快捷键、帮助）**

```go
func TestTUIHelpShowsContextKeys(t *testing.T) {
	m := NewTuiModel(config.Default(), nil)
	m.ready = true
	m.width, m.height = 100, 30
	_, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	v := m.View()
	if !strings.Contains(v, "Tab") || !strings.Contains(v, "Esc") {
		t.Fatalf("expected contextual key hints")
	}
}
```

- [ ] **Step 2: 跑全量测试与构建**

Run: `go test ./... && go build -o starman ./cmd/starman`
Expected: 全部 PASS。

- [ ] **Step 3: 手工 smoke 验证**

Run: `./starman --help`
Expected: 仅出现 `sync/search/config/completion`。

Run: `./starman`
Expected: 进入 TUI，侧栏显示 发现/整理/处理/系统 分组。

- [ ] **Step 4: 文档收口**

```md
### Usage
starman                    # Launch TUI
starman sync               # CLI sync
starman search <query>     # CLI search
starman config init|show
starman completion <shell>
```

- [ ] **Step 5: 提交**

```bash
git add README.md README.zh.md internal/tui/tui_test.go
git commit -m "test/docs: finalize TUI-first redesign acceptance"
```

---

## Self-Review

- **Spec coverage:**
  - CLI 保留/下沉映射：Task 1
  - Action 统一执行层：Task 2-3
  - 任务流导航与交互：Task 4-5
  - TUI 功能真实接线：Task 6-7
  - 测试与文档验收：Task 8
- **Placeholder scan:** 无 `TBD/TODO/implement later` 等占位项。
- **Type consistency:** `SyncOpts/SearchOpts/Result/TaskState` 在任务定义中前后一致。
