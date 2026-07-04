# Starman TUI 重构设计

- 日期：2026-07-04
- 状态：待实施
- 基于：`AGENTS.md` Starman TUI 开发规范 v1.0 + 技术债收敛路线图

## 0. 背景与目标

当前 TUI 实现（`/docs/superpowers/specs/2026-07-04-tui-design.md`）与 `AGENTS.md` 规范存在系统性偏离。本次重构的目标：将 TUI 代码库完全收敛到 `AGENTS.md` 规范，覆盖路线图中全部 P0/P1/P2 技术债。

## 1. 实施方案：按依赖层次逐层推进

```
L0: CLI 基础设施改造 ──► L1: 命令执行链路 ──► L3: 功能补齐 ──► L4: P2 体验改进
                           L2: 组件拆分 + 布局 ──┘
```

每层完成后可独立编译验证，测试边重构边写。

---

## 2. L0：CLI 基础设施改造

### 2.1 root.Run() 返回 error

`internal/cli/root.go` 中 4 处 `os.Exit(1)` 全部改为 `return fmt.Errorf(...)`：

| 行 | 场景 | 改造 |
|---|---|---|
| 49 | 解析 config 路径失败 | `return fmt.Errorf("parse config path: %w", err)` |
| 55 | 加载 config 失败 | `return fmt.Errorf("load config: %w", err)` |
| 60 | TUI 运行失败 | `return fmt.Errorf("tui: %w", err)` |
| 68 | cobra 命令执行失败 | `return err` |

`Run()` 签名：`func Run(ver string)` → `func Run(ver string) error`。

调用方 `cmd/starman/main.go` 改为检查 error 后 `os.Exit(1)`（`os.Exit` 仅保留在 main.go）。

### 2.2 子命令输出走 cmd.OutOrStdout/ErrOrStderr

所有子命令 `RunE` 中 `fmt.Fprintf(os.Stderr, ...)` → `fmt.Fprintf(cmd.ErrOrStderr(), ...)`，`fmt.Fprintf(os.Stdout, ...)` → `fmt.Fprintf(cmd.OutOrStdout(), ...)`。

涉及文件：`sync.go`、`search.go`、`analyze.go`、`generate.go`、`backup/*.go`、`release/*.go`、`category.go`、`config.go`。

### 2.3 每次执行新建 Command 实例

`NewRootCmd(ver)` 当前已符合（每次调用新建实例），无需改动。确认所有测试中同样新建实例，不共享全局命令对象。

### 2.4 L0 测试

- `root.Run()` error 返回路径表驱动测试（config 缺失、TUI 失败）
- 各子命令输出重定向验证：`SetOut`/`SetErr` 到 `bytes.Buffer`，断言输出内容

---

## 3. L1：命令执行链路

### 3.1 cmdrunner — headless cobra runner

新包 `internal/tui/cmdrunner/`：

```
internal/tui/cmdrunner/
├── cmdrunner.go
└── cmdrunner_test.go
```

```go
type Runner struct {
    version string
}

func NewRunner(version string) *Runner

func (r *Runner) Run(ctx context.Context, input string) (stdout, stderr string, err error)
```

核心逻辑：
1. 按 shell 语义切分输入字符串（支持单/双引号参数）
2. `cli.NewRootCmd(r.version)` 新建实例
3. `cmd.SetArgs(args)`、`cmd.SetOut(&outBuf)`、`cmd.SetErr(&errBuf)`
4. `cmd.ExecuteContext(ctx)` 执行
5. 返回 stdout、stderr、error

例外拦截（返回特定哨兵错误）：
- `config init` → `ErrInteractiveRequired`（路由到 Prompt 组件）
- `sync --watch` → `ErrBlockingRequired`（路由到定时 Task）

### 3.2 drawer — 输出抽屉组件

新包 `internal/tui/components/drawer/`：

```
internal/tui/components/drawer/
├── drawer.go
└── drawer_test.go
```

特性：
- 默认收起（高度 0），`Ctrl+O` 展开，高度 = `floor(ScreenHeight * 0.35)`
- 展开后主体高度同步缩小（挤压式）
- 内部 `viewport`：可滚动命令历史，最新在前
- 每条历史：`[14:30:05] :sync --full` + stdout/stderr 文本
- `Esc` 收起（保留历史不清空）、`Ctrl+L` 清空历史
- `Esc` 时若抽屉已展开且聚焦，则关闭；若 drawer 未聚焦，Esc 行为不变

### 3.3 搜索改造 — headless search --json

当前搜索：`ui.go` 中内存调用 `filterRepos()` 过滤全量仓库。

改造后：
1. 搜索触发时调用 `cmdrunner.Run(ctx, "search " + query + " --json")`
2. 解析 JSON 输出更新 `starssection` 的 rows
3. `search.go` 新增 `--json` flag，启用时输出 JSON 而非表格文本

### 3.4 L1 测试

| 组件 | 内容 |
|---|---|
| `cmdrunner` | shell 切分（含引号参数、转义符）、命令执行成功/失败、`config init`/`sync --watch` 例外拦截、`SetOut`/`SetErr` 重定向验证 |
| `drawer` | 展开/收起状态机、`Ctrl+O`/`Esc`/`Ctrl+L` 按键、窄屏高度计算、空历史渲染、3 组尺寸 golden |
| 搜索链路 | `search --json` 端到端（mock store） |

---

## 4. L2：组件拆分 + 响应式布局

### 4.1 SearchInput 组件

新包 `internal/tui/components/searchinput/`：

```
internal/tui/components/searchinput/
├── searchinput.go
└── searchinput_test.go
```

- 独立 `tea.Model`，内嵌输入行
- 状态：`query string`、`focused bool`
- 按键：`Enter` → 发 `SearchExecutedMsg{Query}`；`Esc` → 退出搜索回 Normal
- 视觉：底部单行输入，左侧 🔍 图标

### 4.2 Prompt 组件

新包 `internal/tui/components/prompt/`：

```
internal/tui/components/prompt/
├── prompt.go
└── prompt_test.go
```

统一覆盖层，支持三种模式：

| 模式 | 触发场景 | 内容 |
|---|---|---|
| 确认框 | `sync --full`、`analyze --all`、unstar | 标题 + y/n 提示 |
| 分类选择器 | `c` 键编辑分类 | 列表导航（j/k）+ Enter 确认 |
| 标签编辑器 | `t` 键编辑标签 | 单行输入 `+tag,-tag` |

- 统一按键：`Enter`/`y` 确认、`Esc`/`n` 取消
- 视觉：居中覆盖层，半透明背景遮罩，主区域变暗

### 4.3 ui.go 精简

拆分后 `ui.go` 从 ~1036 行精简到 ~400 行，仅保留：
- Model 结构体（持有组件引用）
- `Update()` 消息路由（按模式转发 KeyMsg）
- `View()` 组装布局
- `recalcLayout()` 响应式计算

### 4.4 响应式布局

修改 `recalcLayout()` 实现三档断点：

| 断点 | 条件 | sidebar | 主表格 |
|---|---|---|---|
| 宽屏 | `ScreenWidth >= 80` | 右侧，宽 = `max(28, floor(w * 0.38))` | `w - sidebarWidth` |
| 窄屏 | `50 <= ScreenWidth < 80` | 底部，高 = `floor(h * 0.4)` | `w`（全宽） |
| 极窄 | `ScreenWidth < 50` | 关闭（强制 `p` off） | `w`（全宽） |

- `PreviewPosition` 支持 `right`/`bottom`/`auto`（auto 根据宽度自动）
- 手动 `P` 键覆盖自动模式
- 各组件 `View()` 接收可用宽高参数，对不足宽度做截断

### 4.5 L2 测试

| 组件 | 内容 |
|---|---|
| `searchinput` | 聚焦/失焦状态机、`Enter`/`Esc` 消息类型、输入截断、3 组 golden |
| `prompt` | 三种模式各自的状态机、按键处理（j/k/Enter/Esc）、边界 clamp（首行 ↑、末行 ↓）、3 组 golden |
| `ui.go` | 消息路由覆盖所有模式切换路径 |
| `recalcLayout` | 三档断点计算正确性、`auto` ↔ 手动切换、sidebar 宽高计算不越界 |
| `sidebar` | right/bottom 两种位置渲染、3 种宽度下无越界 |

---

## 5. L3：功能补齐

### 5.1 Stars 分组 tabs 暴露

`tabs.Model` 新增第二行 section tabs 渲染。Stars 视图 sections：

```
[🔍 搜索] [All] [Language] [Category] [Tag]
```

- 搜索 section 第 0 位固定，走 L1 的 `search --json` 链路
- All/Language/Category/Tag 复用 `group.go` 中已有分组函数
- `]]`/`[[` 在 sections 间切换

### 5.2 键位语法糖

动作键拼命令字符串 + 可选确认 → `cmdrunner.Run()`：

| 按键 | 确认 | 命令字符串 |
|---|---|---|
| `s` | 无 | `sync` |
| `s` | y/n yes | `sync --full` |
| `a` | y/n yes | `analyze --all` |
| `a` | y/n no | `analyze` |
| `x` | 已 star | `unstar owner/repo` |
| `x` | 未 star | `star owner/repo` |
| `c` | 选中分类 | `categorize owner/repo <category>` |
| `t` | 输入标签 | `tag owner/repo +tag,-tag` |

破坏性操作（`--full`、`--all`、unstar）保留 y/n 确认。

### 5.3 补齐命令模式

所有 CLI 命令通过 `cmdrunner` 自动可用，不做额外实现。特殊处理：
- `:config init` → runner 返回 `ErrInteractiveRequired` → 路由到 Prompt 组件
- `:sync --watch` → runner 返回 `ErrBlockingRequired` → 路由到定时 Task

### 5.4 L3 测试

| 内容 |
|---|
| Stars 分组 tabs 的 `]]`/`[[` 切换、各分组数据正确性 |
| 每个按键的"拼命令字符串"逻辑、确认框触发条件 |
| 确认框 y/n 两条路径 |

---

## 6. L4：P2 体验改进

### 6.1 可配置 keybindings

`config.yaml` 新增 `tui.keybindings` 覆盖默认值：

```yaml
tui:
  keybindings:
    universal:
      quit: "q"
      refresh: "r"
      toggle_sidebar: "p"
    stars:
      sync: "s"
      analyze: "a"
      toggle_star: "x"
```

- `keys.NewKeyMap(cfg *config.TUIKeybindings) *KeyMap`：未配置的键位回退默认值
- `KeyMap` 放入 `ProgramContext` 字段，`ui.go` 中通过 `m.ctx.Keys.Quit` 等引用

### 6.2 Light 主题

- 新增 `theme.LightTheme()`，token 与 `DefaultTheme()` 对应但取浅色值
- `config.yaml` 中 `tui.theme: light` 时启用
- 所有组件从 `ctx.Theme` 取值，无需额外改动

### 6.3 glamour README 渲染

- `repoview` README tab：优先显示缓存的 `ReadmeContent`，用 glamour 渲染
- 无缓存时异步获取（`tea.Cmd`），获取后缓存到 `store`
- 需要确认 store 中是否有 `ReadmeContent` 字段；若无则新增

### 6.4 鼠标支持

`ui.go` 的 `Update()` 处理 `tea.MouseMsg`：
- 点击 tabs 切换视图/section
- 点击 sidebar tab 切换
- 点击表格行选中
- 点击 footer viewSwitcher 切换

使用 bubbletea v2 原生鼠标支持（`tea.WithMouseCellMotion()`）。

### 6.5 L4 测试

| 内容 |
|---|
| `keys.NewKeyMap` 覆盖/回退正确性 |
| `LightTheme()` 所有 token 非零值 |
| glamour 渲染（mock ReadmeContent）|
| 鼠标点击各区域触发正确消息 |

---

## 7. 文件变更总览

| 层 | 新增 | 修改 |
|---|---|---|
| L0 | 无 | `cli/root.go`、`cli/sync.go`、`cli/search.go`、`cli/analyze.go`、`cli/generate.go`、`cli/backup/*.go`、`cli/release/*.go`、`cli/category.go`、`cli/config.go`、`cmd/starman/main.go` |
| L1 | `tui/cmdrunner/cmdrunner.go`、`tui/cmdrunner/cmdrunner_test.go`、`tui/components/drawer/drawer.go`、`tui/components/drawer/drawer_test.go` | `ui.go`（搜索链路段）、`cli/search.go`（`--json` flag） |
| L2 | `tui/components/searchinput/searchinput.go`、`tui/components/searchinput/searchinput_test.go`、`tui/components/prompt/prompt.go`、`tui/components/prompt/prompt_test.go` | `ui.go`（拆分精简）、`components/sidebar/sidebar.go`、`context/context.go` |
| L3 | 无 | `tabs/tabs.go`（section tabs 行）、`starssection/starssection.go`、`ui.go`（按键处理改为语法糖）、`commands.go`（移除手写解析，改为 cmdrunner） |
| L4 | `theme/light.go`（如需独立文件）| `keys/keys.go`、`repoview/repoview.go`、`ui.go`（鼠标处理） |

---

## 8. 不变项

- Bubbletea v2 + Lipgloss v2 + Bubbles v2 技术栈
- 业务包接口（store/github/ai/discovery/release/backup）不变
- CLI 子命令的对外行为（参数、flag、输出格式）不变
- 数据库 schema 不变
- Section 接口签名不变（可能小幅扩展）

---

## 9. 风险与缓解

| 风险 | 缓解 |
|---|---|
| CLI 改造导致子命令回归 | 现有 CLI 测试全部保留并通过 |
| cmdrunner shell 切分 bug（引号/转义） | 表驱动测试覆盖常见 shell 模式 |
| 响应式布局中间状态错乱 | L3 golden 测试覆盖 3 组尺寸 |
| 重构范围大、中间编译失败 | 每层完成后 `make build && make test && make lint` |
