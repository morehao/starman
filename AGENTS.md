# Starman 开发规范

## 命令

```bash
make build          # CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath -o starman ./cmd/starman/
make install        # go install -ldflags="-s -w" -trimpath ./cmd/starman/
make test           # go test ./...
make lint           # golangci-lint run
make clean          # rm -f starman
```

运行单个包或测试：
```bash
go test ./internal/store/                         # 单个包
go test -run TestUpsertAndListRepositories ./internal/store/  # 单个测试
```

## 集成测试

集成测试使用 `//go:build integration` 标签。运行方式：
```bash
./scripts/test-integration.sh                    # 全部集成测试
./scripts/test-integration.sh ./internal/github/  # 指定包
```

需要 `.env` 文件配置 `GITHUB_TOKEN` + `GITHUB_USERNAME`。脚本会 source `.env` 然后执行 `go test -v -tags=integration -count=1 -timeout 5m ./...`。

## 架构

- **入口**：`cmd/starman/main.go`（8 行，调用 `cli.Execute()`）
- **无子命令参数时启动 TUI** — 见 `cli/root.go` 中的 `hasSubcommandArgs`
- **包依赖流向**：`cli` / `tui` → 业务包（`github`、`ai`、`generate`、`backup`、`release`、`discovery`）→ `store`，禁止反向依赖，禁止 `tui` 直接 import `cli`
- **纯 Go SQLite**：`modernc.org/sqlite` — 无 CGO，支持交叉编译。数据库使用 WAL 模式，启用外键约束。
- **数据库路径**：`~/.starman/starman.db`，配置文件 `~/.starman/config.yaml`
- **第三方类型不跨包边界泄漏**：`github` 包在内部将 go-github 类型转换为本地 `store` 类型

## 配置优先级

CLI 参数 > 环境变量 > 配置文件 > 默认值。

关键环境变量：`STARMAN_GITHUB_TOKEN`、`STARMAN_AI_API_KEY`、`STARMAN_AI_BASE_URL`、`STARMAN_AI_MODEL`、`STARMAN_EMBEDDING_API_KEY`、`STARMAN_EMBEDDING_BASE_URL`、`STARMAN_EMBEDDING_MODEL`。

## 代码约定

- 标准 Go 格式化。错误使用 `fmt.Errorf("context: %w", err)` 包装。
- 优先使用表驱动测试。测试辅助函数模式：`testStore(t *testing.T)`、`testToken(t *testing.T)` — 调用 `t.Helper()`、`t.Fatal`、`t.Cleanup`。
- 单元测试使用内存 SQLite（`:memory:`）；HTTP 模拟使用 `httptest.NewServer`。
- `store` 中已废弃方法有 `// Deprecated:` godoc 注释。
- SQLite 中 JSON 序列化的切片列：`Topics`、`AITags`、`AIPlatforms`。
- 版本通过 ldflags 注入：`-X github.com/morehao/starman/internal/version.ver=$(git describe --tags --always --dirty)`，默认值 `"dev"`。

## 发布

CI 在 `v*` 标签上触发，运行 `goreleaser release --clean`。产出多平台二进制文件、deb/rpm 包、Homebrew formula、Scoop manifest。

## Git Worktrees

`.worktrees/` 在 `.gitignore` 中 — 项目使用 `git worktree` 进行并行开发。

---

# Starman TUI 开发规范

- 版本：v1.0
- 适用范围：`internal/tui/` 下的所有代码，以及后续所有新增视图/组件/命令
- 技术栈：`charm.land/bubbletea/v2` + `charm.land/lipgloss/v2` + `charm.land/bubbles/v2` + `glamour`
- 定位：本文档是**约束性规范**，不是现状描述。现有实现中偏离本规范的部分，视为技术债，新功能开发**不得沿用**旧的偏离模式。

---

## 1. 设计原则

1. **展示层与业务层分离**：`internal/tui` 只做交互和渲染，所有数据读写通过 `internal/store`、`internal/github`、`internal/ai`、`internal/discovery`、`internal/release`、`internal/backup` 完成。TUI 不应该重新实现一遍 CLI 已有的业务逻辑。
2. **一个交互单元 = 一个组件**：只要是用户能输入/能聚焦/有自己状态的 UI 单元（搜索框、确认框、编辑器、覆盖层），必须是独立的 `tea.Model`，放在 `components/` 下，不允许再内联进 `ui.go`。
3. **窄终端是一等场景**：所有新视图/组件必须在设计阶段就想清楚宽度 < 80、高度 < 20 时如何降级，而不是事后补丁。
4. **能配置的不硬编码**：颜色走 theme token、键位走 keymap 注册表、可开关的行为走 config。
5. **长任务必须走 Task 系统**：任何调用 GitHub API / AI API / 网络 IO 的操作，必须包装成 `tea.Cmd` 通过 Task 机制异步执行，`Update()` 里不允许出现阻塞调用。
6. **命令模式与快捷键覆盖同一组业务能力**：凡是 CLI 有的操作（`sync`/`analyze`/`generate`/`backup`/`tag`/`categorize`/`release ...`），TUI 至少要能通过 `:` 命令模式触达，高频操作再额外给快捷键。

---

## 2. 整体布局规范

### 2.1 骨架

除 Stats 视图（整屏统计，无 sidebar）外，所有视图统一遵循三段式骨架：

```
┌─ Tabs（2 行：视图级 tab + 当前视图内的 section tab）──────────────┐
├─ 主体：主表格（左） + Sidebar 预览（右 或 下）────────────────────┤
├─ Footer（1 行：viewSwitcher + 分页信息 + 任务状态 + ?help 提示）──┘
```

- **Tabs 第 1 行**：logo/版本 + 视图级 tabs（Stars / Trending / Releases / Stats）
- **Tabs 第 2 行**：当前视图内的 section tabs。规则：**第 0 位固定是 🔍 搜索 section**
- **主体**：主表格用 `listviewport` 渲染，宽度 = `ctx.MainContentWidth`；sidebar 用统一的 `sidebar.Model` 容器
- **Footer**：固定 1 行，新视图不允许自定义 footer 结构，只能通过 Task 系统上报状态

### 2.2 响应式规则（强制要求）

| 断点 | 宽度条件 | 行为 |
|---|---|---|
| 宽屏 | `ScreenWidth >= 80` | sidebar 位于右侧，宽度 = `max(28, floor(ScreenWidth * 0.38))` |
| 窄屏 | `ScreenWidth < 80` | sidebar 自动切换到底部，高度 = `floor(ScreenHeight * 0.4)` |
| 极窄 | `ScreenWidth < 50` | 关闭 sidebar（`p` 键状态强制为 off），仅保留主表格 + 精简 footer |

任何新组件在实现 `View()` 时，**必须**接收当前可用宽高作为参数，并对"可用宽度小于组件最小宽度"的情况有显式处理（截断文字加 `…`、隐藏次要列、换行），不允许假设终端足够宽从而导致布局错乱或 panic。

### 2.3 视图定义规则

新增视图时必须回答以下问题：

| 问题 | 说明 |
|---|---|
| 数据源是哪个业务包？ | 必须是 `internal/` 下现有或新增的业务包，禁止在 TUI 层直接拼 SQL / 直接调 REST |
| 有没有多个查看维度？ | 有 → 用 section tabs 表达；没有 → 单 section |
| 有没有 sidebar 详情？ | 有 → 复用 `repoview`/`releaseview`，或新增；没有 → 显式声明 |
| 分页/懒加载策略？ | 实现 `Section.FetchNextPageSectionRows()`，不要一次性拉全量到内存 |
| 空态文案？ | 必须提供 |
| 窄屏降级策略？ | 按 2.2 的断点表实现，不能省略 |

### 2.4 空态与错误展示规范

- **空态**：主区域垂直居中，格式固定为「图标 + 一句话结论 + 1-2 条操作建议（带高亮的快捷键）」
- **错误条**：统一用 `renderErrorBar()`，渲染在主区域下方、footer 上方，单行，5 秒自动清除
- **致命错误**：整屏样式覆盖，明确报错原因 + 补救建议，`q` 退出

---

## 3. 视觉与主题规范

### 3.1 颜色必须走 theme token，禁止硬编码

- 所有颜色引用必须来自 `theme.Theme` 结构体的语义化字段（如 `theme.Primary`、`theme.Muted`、`theme.Danger`、`theme.Success`），**不允许**在组件代码里直接写 `lipgloss.Color("#xxxxxx")`。

### 3.2 图标与间距

- 复用 `constants/` 里已有的图标常量，新视图需要新图标时先加到 `constants/`
- 表格列间距、sidebar 内边距等复用 `common/styles.go` 里的既有样式

### 3.3 组件视觉一致性检查表

新组件提交前自查：
- [ ] 颜色全部来自 `theme` token
- [ ] 图标全部来自 `constants`
- [ ] 选中行/聚焦态视觉表现与其他视图一致（选中行整行高亮背景 + `▸` 前缀）
- [ ] 加载态统一用现有 spinner 组件
- [ ] 在 80x24、120x40、50x20 三种终端尺寸下验证布局不错位、不越界

---

## 4. 交互与键位规范

### 4.1 键位分层

全部注册在 `keys/` 包，**不允许**在 `ui.go` 或具体 section 里出现裸的 `case "x":` 分支处理业务逻辑：

| 层级 | 范围 | 示例 |
|---|---|---|
| 通用层 | 所有视图都生效 | `j/k` 上下、`g/G` 首末行、`]]`/`[[` 切 section、`Tab` 切视图、`/` 搜索、`:` 命令模式、`?` 帮助、`q` 退出 |
| 视图层 | 仅在某个视图下生效 | Stars 的 `s`(sync)/`a`(analyze)/`x`(star/unstar)；Releases 的 `s`/`u`/`p` |
| 组件层 | 仅在某个聚焦的子组件里生效 | 搜索输入框里的 `Enter`/`Esc`，overlay 编辑器里的 `Tab` 切字段 |

**新增键位的强制流程**：
1. 在 `keys/keys.go` 的对应层级里注册按键与描述文案
2. 在 help 面板（`?`）里补上对应条目
3. 如果是高频操作，评估是否与既有键位冲突
4. 低频、带参数的操作**优先**放进命令模式

### 4.2 交互模式状态机

```
Normal(表格导航) ──/──► Search ──Enter──► 执行搜索 ──► 回 Normal
       │
       ├──:──► Command ──Enter──► 解析并执行 ──► 回 Normal
       │
       ├──[动作键 s/a/t/c/x 等]──► 若需要参数/确认 → Prompt(overlay) ──► 执行 ──► 回 Normal
       │                        └─ 若无需参数 → 直接触发 Task ──► 回 Normal
       │
       └──[j/k/g/G/h/l 等]──► 路由给 currSection.Update()
```

规则：
- **Search / Command / Prompt 都必须是独立组件**，实现统一的"聚焦态"接口（`IsFocused() bool`），`ui.go` 的顶层 `Update()` 只负责按当前模式把 `KeyMsg` 转发给对应组件
- 任意模式下 `Esc` 必须能无副作用地回到 Normal
- 模式切换时 sidebar 内容不清空

### 4.3 命令模式规范

语法：`:<command> [位置参数...] [--flag] [--key=value]`

**强制要求**：命令模式支持的命令集合，必须与 CLI 子命令集合保持同构映射关系：

| CLI 命令 | TUI 命令模式 |
|---|---|
| `starman sync [--full]` | `:sync [--full]` |
| `starman analyze [--all] [--limit N] [--force]` | `:analyze [--all] [--limit N] [--force]` |
| `starman generate -s ... -o ... --repo ...` | `:generate [-s ...] [-o ...] [--repo ...]` |
| `starman backup webdav --push/--pull/--test` | `:backup webdav --push` 等 |
| `starman tag <repo> +a,-b` | `:tag <repo> +a,-b` |
| `starman categorize <repo> <cat> [--lock]` | `:categorize <repo> <cat> [--lock]` |
| `starman release subscribe/unsubscribe/pull` | `:release subscribe/unsubscribe/pull ...` |
| `starman config show` | `:config show` |
| `starman star/unstar <repo>` | `:star`/`:unstar <repo>` 或直接 `x` 键 |

**禁止事项**：命令模式内部**不得**反向调用 `internal/cli` 里的 cobra Command，必须直接调用 cli 命令背后的业务函数。如果某个 CLI 命令的逻辑深嵌在 `internal/cli` 包内，**先把逻辑下沉成 `internal/<domain>` 的可导出函数，cli 和 tui 共同调用它**。

### 4.4 长文本/结构化输出的展示规范

`:config show`、`:release list` 等产出较长结构化文本的命令，使用 sidebar 容器弹出「输出面板」模式（只读、可滚动的 `viewport`，独立组件），`Esc` 关闭返回 Normal。不允许把大段文本塞进错误条或 footer。

### 4.5 异步任务规范

- 任何调用外部 API / 长耗时操作，一律通过 `ctx.StartTask(Task{...})` 发起，完成时发 `TaskFinishedMsg`
- Task 状态展示只出现在 footer 右侧：
  - 进行中：`⠏ <动词进行时>...`
  - 成功：`✅ <结果摘要>`，2 秒后自动清除
  - 失败：`❌ <失败摘要>`，保留到下一次操作覆盖
- 批量任务必须支持"部分失败不阻断整体"
- `Update()` 函数体内**不允许**出现同步网络调用、同步数据库长查询

---

## 5. 测试规则

### 5.1 分层策略

```
L4 集成测试        端到端：fake store/github → 启动 → 导航 → 命令 → 断言最终状态
L3 Golden 快照测试  View() 渲染结果的视觉回归
L2 组件测试        直接调用 Update()/View()，不启动完整 tea.Program
L1 纯逻辑单测      分组、解析、状态机转换等不依赖 tea.Model 的函数
```

每一层都是新代码合入的必要条件。

### 5.2 L2 组件测试强制用例清单

每个新 Section/组件都要覆盖：
- 每一个绑定的按键都要有至少一个转换测试
- 光标/选中项在边界处的 clamp 行为（首行按 ↑、末行按 ↓ 不越界）
- `isLoading` 状态切换是否正确反映到 `View()`
- 聚焦态组件的 `Esc` 是否正确退出且不残留状态
- **禁止阻塞检查**：给 `Update()` 传入触发异步操作的按键消息，断言返回值里包含非 nil 的 `tea.Cmd`

### 5.3 L3 Golden 快照测试

每个 Section / 每个独立组件，至少准备 **3 组尺寸**的 golden 用例：宽屏（120x40）、标准（80x24）、窄屏（50x20）。

```go
func TestStarsSection_View_Default(t *testing.T) {
    m := newTestStarsSection(fixtureRepos())
    m.SetSize(120, 40)
    got := stripANSI(m.View())
    golden.RequireEqual(t, []byte(got))
}
```

规则：
- 布局类用例统一 `stripANSI` 后比对，配色/高亮相关的用例保留 ANSI
- CI 中固定颜色 profile（`lipgloss.SetColorProfile(termenv.Ascii)`），避免本地和 CI 终端能力不一致
- 首次新增用 `go test ./... -update` 生成基线
- PR 若改动 golden 文件，必须在 PR 描述里说明"这是预期的视觉变更"

### 5.4 L4 集成测试关键旅程

| # | 旅程 | 覆盖点 |
|---|---|---|
| 1 | 启动 → 无数据 → 空态 → `s` 触发 sync → Task 完成 → 表格填充 | Task 生命周期 + 空态转换 |
| 2 | 导航（j/k）→ sidebar 联动更新 → 切 sidebar tab（h/l） | 跨组件状态同步 |
| 3 | `/` 搜索 → 输入 → 回车 → 结果集过滤 → `Esc` 恢复原列表 | 搜索模式完整闭环 |
| 4 | `:sync --full` → 命令解析 → Task 触发 → 成功/失败两条分支 | 命令模式 + Task 集成 |
| 5 | 窄屏/极窄尺寸下启动 → sidebar 响应式切换 | 响应式规则端到端验证 |
| 6 | 批量任务部分失败 → footer 正确显示进度与失败数 | 批量任务失败隔离 |

使用 fake `store`/`github`/`discovery` 实现，异步断言用 `teatest.WaitFor` 轮询而不是 `time.Sleep`。

### 5.5 新增视图/组件的测试 Checklist

合入前逐项确认：
- [ ] L1：所有纯函数有表驱动测试，覆盖空/单/边界输入
- [ ] L2：每个按键绑定至少一个 `Update()` 转换测试
- [ ] L2：光标/滚动边界 clamp 测试
- [ ] L2：异步操作触发处返回非阻塞 `tea.Cmd`
- [ ] L3：宽屏/标准/窄屏三组 golden 快照
- [ ] L3：布局与样式类用例分开
- [ ] L4：至少一条覆盖该功能的端到端旅程
- [ ] 空态、错误态有对应用例
- [ ] 若涉及命令模式：`parseCommand` 覆盖新命令的所有 flag 组合

---

## 6. 架构与代码组织约束速查表

| 约束 | 内容 |
|---|---|
| 依赖方向 | `cli`/`tui` → 业务包 → `store`，禁止反向依赖 |
| Section 契约 | 新视图必须实现 `components/section.Section` 接口 |
| 组件目录 | 新组件放 `components/<name>/`，不得散落在 `ui.go` |
| 主题/图标/样式 | 只能引用 `theme/`、`constants/`、`common/` |
| 键位注册 | 只能在 `keys/` 里定义，按三层分类 |
| 命令解析与执行 | 只能在 `commands.go`，禁止反向调用 `internal/cli` |

---

## 7. 用 LLM 辅助开发 TUI 新功能时的提示词要点

```markdown
## 项目背景
- Go 1.22+，charm.land/bubbletea/v2 + charm.land/lipgloss/v2 + charm.land/bubbles/v2
- 严格遵循 Elm 架构，参考迁移指南：
  https://github.com/charmbracelet/bubbletea/blob/main/UPGRADE_GUIDE_V2.md
- 本次改动必须遵守《starman TUI 开发规范》：
  - 交互单元必须是独立 tea.Model，不得内联进 ui.go
  - 颜色只能用 theme 包里的语义化 token，不允许硬编码 lipgloss.Color
  - 新键位必须注册进 keys/ 包并归入正确的层级
  - 长耗时操作必须通过 Task 系统（ctx.StartTask + TaskFinishedMsg），Update() 内禁止阻塞
  - 必须实现宽屏/窄屏/极窄三档响应式行为
  - 命令模式新增命令要和对应 CLI 子命令的参数保持一致

## 功能描述
[状态 / 按键与事件 / 视觉反馈]

## 输出要求
- 完整可编译代码
- 附带 L1+L2+L3 测试
```

---

## 8. 技术债收敛路线图（按优先级排序）

**P0（违反核心原则，优先处理）**
1. 将 `ui.go` 中内联的搜索输入、分类/标签 overlay 拆成独立组件
2. 搜索从内存过滤升级为 `store.Search` 三段式检索
3. 补齐窄屏自动切 bottom 预览 + 极窄关闭 sidebar

**P1（功能完整性缺口）**
4. 补齐命令模式：`:analyze`/`:generate`/`:backup`/`:tag`/`:categorize`/`:release ...`/`:config show`
5. Stars 视图暴露分组 tabs（Language/Category/Tag）
6. 新增"输出面板"组件，承接结构化输出需求

**P2（体验与可维护性）**
7. keybindings 从 `config.yaml` 的 `tui.keybindings` 读取覆盖默认值
8. Light 主题
9. README tab 用 glamour 渲染原始 Markdown
10. 鼠标支持
