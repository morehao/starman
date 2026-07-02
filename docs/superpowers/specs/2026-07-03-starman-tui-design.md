# Starman TUI 技术设计文档

## 1. 概述

为 starman 新增交互式 TUI 模式，使用 `bubbletea` + `lipgloss` + `bubbles` 技术栈。通过 `starman tui` 子命令启动，与现有 CLI 命令并存。

## 2. CLI/TUI 分工

判断标准：**核心动作是"浏览/选择/查看"则 TUI 化，"单次执行/批处理/管道输出"则保留 CLI**。

### 命令全集一览

| 命令 | 子命令 | 归属 | 说明 |
|------|--------|------|------|
| `sync` | — | **CLI** | 批处理，`--watch` daemon，stdout 干净 |
| `analyze` | — | **CLI** | 批量后台任务，CI 可调用 |
| `generate` | — | **CLI** | 输出 Markdown 到 stdout/文件，`--repo` 自动推送 |
| `backup` | `json`, `webdav` | **CLI** | 导入导出/WebDAV 定时同步 |
| `star` | — | **CLI** | 单次操作，一行命令最快 |
| `unstar` | — | **CLI** | 同上 |
| `config` | `init`, `show` | **CLI** | 问答式交互已够用；TUI 启动依赖已存在的配置 |
| `completion` | — | **CLI** | 生成 shell 脚本，纯管道用途 |
| `release` | `pull`, `subscribe`, `unsubscribe` | **CLI** | 单次/批处理操作 |
| `release` | `list` | **TUI** | 浏览未读 release，标记已读 |
| `trending` | — | **TUI** | 浏览列表 + 多选收藏；砍掉原 CLI（体验差） |
| `search` | — | **TUI** | 结果列表 + 实时过滤 + 详情跳转；脚本场景用 `--json` 逃生舱 |
| `info` | — | **TUI** | 详情展示 + README 滚动 + 变体切换 |
| `stats` | — | **TUI** | 多维度切换仪表盘 |
| `tag` | 单仓库模式 | **CLI** | `starman tag owner/repo +tag1,-tag2` 单次操作 |
| `tag` | 批量模式 | **TUI** | 勾选仓库再批量编辑 |
| `categorize` | 单仓库模式 | **CLI** | `starman categorize owner/repo frontend --lock` 单次操作 |
| `categorize` | 批量模式 | **TUI** | 勾选仓库再批量设置分类/锁定 |

> 注：`categorize` 命令代码已实现（`internal/cli/categorize.go`），需在 `root.go` 中补注册 `root.AddCommand(newCategorizeCmd())`。

## 3. 技术选型

| 组件 | 库 | 用途 |
|------|-----|------|
| TUI 框架 | `github.com/charmbracelet/bubbletea` | Elm 架构，Model/Update/View |
| 样式 | `github.com/charmbracelet/lipgloss` | 边框、颜色、内边距、对齐 |
| 组件库 | `github.com/charmbracelet/bubbles` | table, textinput, viewport, spinner, help, progress, paginator |

所有依赖均为 charmbracelet 生态，不引入第三方 UI 库。

## 4. 架构设计

### 4.1 包结构

```
internal/
├── tui/                          # 新增 TUI 包
│   ├── tui.go                    # TuiModel 顶层，tea.Program 入口
│   ├── messages.go               # 全局消息类型定义
│   ├── styles/
│   │   ├── theme.go              # 全局色彩/字体主题
│   │   └── components.go         # 组件样式（按钮/卡片/表格/输入框）
│   ├── components/               # 可复用 UI 组件
│   │   ├── sidebar.go            # 侧栏导航
│   │   ├── statusbar.go          # 底部状态栏
│   │   ├── table.go              # 通用数据表格（封装 bubbles/table）
│   │   ├── input.go              # 输入框组件
│   │   ├── confirm.go            # 确认对话框
│   │   ├── spinner.go            # 加载动画
│   │   └── help.go               # 快捷键帮助面板
│   └── pages/                    # 各功能页面（每个实现 tea.Model）
│       ├── dashboard.go          # 仪表盘
│       ├── repo_list.go          # 仓库列表
│       ├── repo_detail.go        # 仓库详情
│       ├── search.go             # 搜索
│       ├── trending.go           # Trending 浏览
│       ├── sync.go               # 同步（进度 + 结果）
│       ├── analyze.go            # 分析（进度 + 结果）
│       ├── tag.go                # 标签管理
│       ├── categorize.go         # 分类管理
│       ├── release.go            # Release 浏览
│       ├── generate.go           # Awesome List 生成
│       ├── backup.go             # 备份/恢复
│       ├── stats.go              # 统计
│       └── config.go             # 配置管理
└── cli/
    └── tui.go                    # `starman tui` cobra 命令
```

### 4.2 嵌套 Model 架构

```
┌─ TuiModel (顶层，实现 tea.Model) ───────────────────────────┐
│                                                              │
│  ┌─ Sidebar ────────────┐  ┌─ ContentArea ────────────────┐ │
│  │  Dashboard       1   │  │                               │ │
│  │  Search         /    │  │  当前选中的页面 Model           │ │
│  │  Repo List      r    │  │  (dashboard / search /        │ │
│  │  Trending       t    │  │   repo_detail / stats / ...)  │ │
│  │  ── 操作 ──          │  │                               │ │
│  │  Sync           s    │  │                               │ │
│  │  Analyze        a    │  │                               │ │
│  │  ── 管理 ──          │  │                               │ │
│  │  Tag            g    │  │                               │ │
│  │  Categorize     c    │  │                               │ │
│  │  Stats          S    │  │                               │ │
│  │  ── 工具 ──          │  │                               │ │
│  │  Release        R    │  │                               │ │
│  │  Generate       G    │  │                               │ │
│  │  Backup         b    │  │                               │ │
│  │  Config         C    │  │                               │ │
│  │  ─────────────────   │  │                               │ │
│  │  Help           ?    │  │                               │ │
│  │  Quit           q    │  │                               │ │
│  └──────────────────────┘  └───────────────────────────────┘ │
│                                                              │
│  ┌─ StatusBar ─────────────────────────────────────────────┐ │
│  │  🔄 Synced just now | ⭐ 328 repos | 🤖 120/500    ?:Help│ │
│  └─────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

### 4.3 数据流

```
TuiModel (顶层调度) ── 接收全局键盘事件，切换页面
    │
    ├── Sidebar: 渲染导航菜单，发送 Navigated 消息
    │
    ├── ContentArea: 根据 pageID 渲染对应子 Model
    │      │
    │      └── 各页面 Model: 实现 tea.Model 接口
    │             │
    │             └── 调用 store.Store 接口获取数据
    │                    │
    │                    └── SQLite (modernc.org/sqlite)
    │
    └── StatusBar: 渲染底部状态
```

**原则**：
1. 单向依赖：页面 → Store 接口，不直接访问文件/网络
2. 异步任务：耗时操作通过 `tea.Cmd` 异步执行，`TaskProgress/TaskCompleted` 回传
3. 页面隔离：页面间通过顶层 `Navigated` 消息调度，不互相调用
4. Store 实例由顶层创建，构造函数注入各页面 Model

### 4.4 全局消息类型

```go
type PageID int

const (
    PageDashboard PageID = iota
    PageSearch
    PageRepoList
    PageTrending
    PageSync
    PageAnalyze
    PageTag
    PageCategorize
    PageStats
    PageRelease
    PageGenerate
    PageBackup
    PageConfig
    PageRepoDetail
)

// 导航
type NavigatedMsg struct{ Page PageID }

// 状态栏消息
type StatusMsg struct {
    Text    string
    Level   StatusLevel // Info/Success/Warning/Error
    Timeout time.Duration
}

// 异步任务
type TaskStartedMsg  struct{ ID, Label string }
type TaskProgressMsg struct{ ID string; Current, Total int }
type TaskDoneMsg     struct{ ID string; Err error }
```

## 5. 页面设计

### 5.1 Sidebar（侧栏导航）

- 14 个菜单项 + Help + Quit，分组（浏览/操作/管理/工具）+ 分隔线
- 当前选中项反色高亮
- 右上角显示快捷键字母
- `↑/↓` 切项，`Enter` 确认，数字键 `1-9` 快速跳转

### 5.2 Dashboard（仪表盘）

```
┌──────────────────────────────────────────────┐
│  📊 Dashboard                                │
│──────────────────────────────────────────────│
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌────┐│
│  │  ⭐ 328  │ │  📅 12  │ │  🤖 5   │ │ ⚠️ ││
│  │ 仓库总数 │ │ 新增(周) │ │ 待分析  │ │ 23 ││
│  └─────────┘ └─────────┘ └─────────┘ └────┘│
│                                              │
│  📊 语言分布 (Top 5)                         │
│  Go      ████████████████  45%  148           │
│  Python  ██████████        28%   92           │
│  Rust    ██████            17%   56           │
│  TS      ███               8%    26           │
│                                              │
│  📋 最近活动                                 │
│  +5 new repos | last sync: 2026-07-03 14:32  │
│  last analyze: 2026-07-03 14:35 (OK)         │
│                                              │
│  🔥 最近 Star 的仓库                         │
│  charmbracelet/bubbletea          ★ 12.3k    │
│  go ⬩ terminal ⬩ framework                  │
└──────────────────────────────────────────────┘
```

### 5.3 Search（搜索页）

- 搜索框：实时 debounce 300ms，支持 `/` 全局快捷键直达
- 模式切换：向量搜索 / LLM 语义 / 文本匹配（Tab 切换焦点）
- 过滤器：语言/分类下拉选择，排序方式切换
- 结果列表：分页表格（bubbles/table）
- `Enter` 查看详情跳转到 repo_detail 页面
- 复用 `ai.Service.Search()` 三层降级策略

### 5.4 Repo Detail（仓库详情）

- Header：名称、星数、标签、AI 摘要、GitHub 链接、基本统计
- AI Summary 面板：完整 AI 分析结果
- README 预览：bubbles/viewport 支持长文本滚动
- Custom 面板：标签编辑（`+add`）、分类下拉（含锁定切换）
- `s` 一键 Star/Unstar，`t` 编辑标签，`c` 切换分类

### 5.5 Repo List（仓库列表）

- 全量仓库表格：名称、语言、星数、分类、自定义标签、同步时间
- 排序切换：星数 / 更新时间 / 名称
- 过滤：语言、分类（复用现有 store 查询过滤）
- `Enter` 进入详情

### 5.6 Trending（趋势浏览）

- RSS / Search API 双源切换
- 仓库列表 + 描述预览
- `s` 一键 Star，`Enter` 查看详情

### 5.7 Sync（同步页面）

- 实时进度条（`TaskProgressMsg` 驱动）
- 同步完成后展示新增仓库列表
- 支持 `--watch` 的定时同步状态展示

### 5.8 Analyze（分析页面）

- 批量分析进度（信号量控制并发度）
- 失败项标红，可重试
- 完成后可选是否重建 FTS5 索引

### 5.9 Tag / Categorize（批量标签/分类管理）

- 仓库列表 + 多选（空格勾选）
- 按语言/分类过滤仓库
- 批量编辑标签（`+tag1  -tag2`）
- 批量设置/锁定分类

### 5.10 Stats（统计）

- 三个 Tab 切换维度：language / category / tag
- 分布条图（字符渲染）
- 仓库数量 + 占比
- 支持 `--json` 导出

### 5.11 Release（Release 浏览）

- 已订阅仓库列表 + 未读 release 数
- 点击展开 release 详情（版本号、日期、changelog）
- mark read / unsubscribe 操作

### 5.12 Generate（生成页面）

- 选择输出模式：language / category / flat
- 预览窗口（viewport 可滚动）
- 确认后生成并可选推送到 GitHub

### 5.13 Backup（备份/恢复）

- 选择操作：JSON 导出/导入 / WebDAV push/pull
- 文件路径输入
- 导入时选择 merge/replace 模式
- 确认对话框（操作带有破坏性）

### 5.14 Config（配置管理）

- 当前配置只读展示（key-value 表格）
- 敏感字段脱敏（token/key/password 显示 `***`）
- 支持编辑特定字段（AI base_url、GitHub token 等）

### 5.15 StatusBar（底部状态栏）

- 左段：同步时间、仓库总数、AI 额度
- 中间：临时消息（3 秒自动清除），按 `StatusMsg.Level` 显示不同颜色
- 右段：快捷键提示 `?:Help  Ctrl+C:Quit`

## 6. 全局快捷键

| 快捷键 | 功能 | 作用域 |
|--------|------|--------|
| `q` | 退出 TUI | 全局 |
| `?` | 切换帮助面板 | 全局 |
| `/` | 跳转搜索页 | 全局 |
| `Esc` | 返回上一页 / 取消 | 全局 |
| `Tab` | 切换面板焦点 | 页面内 |
| `1-9` | 快速跳转 sidebar 前 9 项 | 全局 |
| `↑/↓/j/k` | 列表/菜单项上下移动 | 全局 |
| `Enter` | 确认选择/进入 | 全局 |
| `Ctrl+C` | 强制退出 | 全局 |

## 7. 依赖注入

```go
// TuiModel 的构造函数
func NewTuiModel(cfg *config.Config) *TuiModel {
    store := store.New(cfg) // 复用现有 store 初始化
    return &TuiModel{
        config: cfg,
        store:  store,
        sidebar:   components.NewSidebar(),
        statusbar: components.NewStatusBar(),
        pages: map[PageID]tea.Model{
            PageDashboard:  pages.NewDashboard(store),
            PageSearch:     pages.NewSearch(store, cfg.AI),
            PageRepoList:   pages.NewRepoList(store),
            PageRepoDetail: pages.NewRepoDetail(store, cfg.GitHub.Token),
            // ...
        },
    }
}
```

## 8. 测试策略

| 层级 | 测试方式 | 覆盖 |
|------|----------|------|
| `components/*` | 单元测试 | View() 输出正确性，Update() 状态转换 |
| `pages/*` | 集成测试 | 通过 `tea.NewProgram()` + `testModel` 验证状态流转 |
| `tui.go` | 冒烟测试 | 确保 Model 初始化不 panic，各页面可切换 |
| `styles/*` | 快照测试 | 固定输入 → 固定 lipgloss 输出，防回归 |

## 9. 实施计划

| 阶段 | 任务 | 预计产出 |
|------|------|----------|
| **Phase 1** | 基础设施 | `tui.go` 顶层框架、Sidebar、StatusBar、主题系统、消息定义 |
| **Phase 2** | 核心页面 | Dashboard、RepoList、RepoDetail、Search |
| **Phase 3** | 操作页面 | Trending、Sync、Analyze、Tag、Categorize |
| **Phase 4** | 工具页面 | Stats、Release、Generate、Backup、Config |
| **Phase 5** | 全局功能 | Help 面板、确认对话框、`starman tui` cobra 命令、测试 |
